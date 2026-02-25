package jobqueue

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type QStashPublisherConfig struct {
	BaseURL          string
	Token            string
	TargetBaseURL    string
	Retries          int
	InternalJobToken string
	Timeout          time.Duration
}

type QStashPublisher struct {
	client           *http.Client
	baseURL          string
	token            string
	targetBaseURL    string
	retries          int
	internalJobToken string
	logger           *slog.Logger
}

func NewQStashPublisher(cfg QStashPublisherConfig, logger *slog.Logger) *QStashPublisher {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &QStashPublisher{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL:          strings.TrimRight(cfg.BaseURL, "/"),
		token:            strings.TrimSpace(cfg.Token),
		targetBaseURL:    strings.TrimRight(strings.TrimSpace(cfg.TargetBaseURL), "/"),
		retries:          cfg.Retries,
		internalJobToken: strings.TrimSpace(cfg.InternalJobToken),
		logger:           logger,
	}
}

func (p *QStashPublisher) Enqueue(ctx context.Context, path string, payload any, delay time.Duration, deduplicationID string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("job path is required")
	}

	targetURL := p.targetBaseURL + path
	publishURL := p.baseURL + "/v2/publish/" + url.QueryEscape(targetURL)
	bodyPayload := payload
	if bodyPayload == nil {
		bodyPayload = map[string]any{}
	}

	body, err := jsoniter.Marshal(bodyPayload)
	if err != nil {
		return fmt.Errorf("marshal job payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, publishURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create qstash request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Upstash-Method", http.MethodPost)
	if p.retries > 0 {
		req.Header.Set("Upstash-Retries", fmt.Sprintf("%d", p.retries))
	}
	if delay > 0 {
		req.Header.Set("Upstash-Delay", normalizeDelay(delay))
	}
	if strings.TrimSpace(deduplicationID) != "" {
		req.Header.Set("Upstash-Deduplication-Id", strings.TrimSpace(deduplicationID))
	}
	if p.internalJobToken != "" {
		req.Header.Set("Upstash-Forward-X-Internal-Job-Token", p.internalJobToken)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("publish qstash job: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("publish qstash job status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	p.logger.InfoContext(ctx, "qstash job published", "path", path, "delay", normalizeDelay(delay), "deduplication_id", deduplicationID)
	return nil
}

func normalizeDelay(delay time.Duration) string {
	if delay <= 0 {
		return "0s"
	}
	seconds := int(delay.Round(time.Second).Seconds())
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%ds", seconds)
}
