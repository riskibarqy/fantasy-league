package observability

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

	syncer := newBetterStackWriteSyncer(
		endpoint,
		strings.TrimSpace(cfg.BetterStackToken),
		cfg.BetterStackTimeout,
	)

	betterStackCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(syncer),
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

	return logger, func(ctx context.Context) error {
		drainCtx := ctx
		if drainCtx == nil {
			drainCtx = context.Background()
		}
		if _, hasDeadline := drainCtx.Deadline(); !hasDeadline {
			withTimeout, cancel := context.WithTimeout(drainCtx, 5*time.Second)
			defer cancel()
			drainCtx = withTimeout
		}
		if err := syncer.Close(drainCtx); err != nil {
			return fmt.Errorf("drain betterstack queue: %w", err)
		}
		if err := logger.Sync(); err != nil && !isIgnorableLoggerSyncError(err) {
			return err
		}
		return nil
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
	endpoint  string
	token     string
	client    *http.Client
	queue     chan []byte
	queueMu   sync.RWMutex
	closeOnce sync.Once
	closed    atomic.Bool
	wg        sync.WaitGroup
	dropped   atomic.Uint64
}

func newBetterStackWriteSyncer(endpoint, token string, timeout time.Duration) *betterStackWriteSyncer {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	s := &betterStackWriteSyncer{
		endpoint: endpoint,
		token:    token,
		client: &http.Client{
			Timeout: timeout,
		},
		queue: make(chan []byte, 1024),
	}
	s.wg.Add(1)
	go s.run()

	return s
}

func (s *betterStackWriteSyncer) Write(p []byte) (int, error) {
	payload := bytes.TrimSpace(p)
	if len(payload) == 0 {
		return len(p), nil
	}

	s.queueMu.RLock()
	defer s.queueMu.RUnlock()
	if s.closed.Load() {
		return len(p), nil
	}

	// Copy payload because zap reuses internal buffers after Write returns.
	copied := make([]byte, len(payload))
	copy(copied, payload)

	select {
	case s.queue <- copied:
	default:
		dropped := s.dropped.Add(1)
		if dropped == 1 || dropped%100 == 0 {
			fmt.Fprintf(os.Stderr, "betterstack queue full; dropped logs=%d\n", dropped)
		}
	}

	return len(p), nil
}

func (s *betterStackWriteSyncer) run() {
	defer s.wg.Done()

	for payload := range s.queue {
		s.send(payload)
	}
}

func (s *betterStackWriteSyncer) send(payload []byte) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "betterstack create request failed: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "betterstack send log failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= http.StatusMultipleChoices {
		fmt.Fprintf(os.Stderr, "betterstack send log got non-2xx status=%d\n", resp.StatusCode)
	}
}

func (s *betterStackWriteSyncer) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.closeOnce.Do(func() {
		s.queueMu.Lock()
		s.closed.Store(true)
		close(s.queue)
		s.queueMu.Unlock()
	})

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *betterStackWriteSyncer) Sync() error {
	return nil
}

func isIgnorableLoggerSyncError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "bad file descriptor") || strings.Contains(msg, "invalid argument")
}
