package observability

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/config"
	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitBetterStackLogger configures logger fanout to stdout and optional Better Stack.
func InitBetterStackLogger(cfg config.Config, baseLogger *logging.Logger) (*logging.Logger, func(context.Context) error, error) {
	if baseLogger == nil {
		baseLogger = logging.NewJSON(cfg.LogLevel)
	}

	if !cfg.BetterStackEnabled {
		baseLogger.Info("betterstack disabled", "reason", "BETTERSTACK_ENABLED=false")
		return baseLogger, func(context.Context) error { return nil }, nil
	}

	endpoint := normalizeBetterStackEndpoint(cfg.BetterStackEndpoint)
	if endpoint == "" {
		return nil, nil, fmt.Errorf("betterstack endpoint cannot be empty")
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		cfg.LogLevel,
	)

	betterStackCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(&betterStackWriteSyncer{
			endpoint: endpoint,
			token:    strings.TrimSpace(cfg.BetterStackToken),
			client: &http.Client{
				Timeout: cfg.BetterStackTimeout,
			},
		}),
		cfg.BetterStackMinLevel,
	)

	zapLogger := zap.New(
		zapcore.NewTee(stdoutCore, betterStackCore),
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	logger := logging.FromZap(zapLogger)
	logger.Info("betterstack enabled",
		"endpoint", endpoint,
		"min_level", cfg.BetterStackMinLevel.String(),
		"service_name", cfg.ServiceName,
		"environment", cfg.AppEnv,
	)

	return logger, func(context.Context) error {
		return logger.Sync()
	}, nil
}

func normalizeBetterStackEndpoint(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return "https://" + value
}

type betterStackWriteSyncer struct {
	endpoint string
	token    string
	client   *http.Client
}

func (s *betterStackWriteSyncer) Write(p []byte) (int, error) {
	payload := bytes.TrimSpace(p)
	if len(payload) == 0 {
		return len(p), nil
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "betterstack create request failed: %v\n", err)
		return len(p), nil
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "betterstack send log failed: %v\n", err)
		return len(p), nil
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= http.StatusMultipleChoices {
		fmt.Fprintf(os.Stderr, "betterstack send log got non-2xx status=%d\n", resp.StatusCode)
	}

	return len(p), nil
}

func (s *betterStackWriteSyncer) Sync() error {
	return nil
}

