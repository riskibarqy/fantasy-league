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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	path = "/" + strings.TrimLeft(strings.TrimSpace(path), "/")
	if strings.TrimSpace(path) == "/" {
		return fmt.Errorf("job path is required")
	}

	baseURL, err := validateHTTPBaseURL(p.baseURL)
	if err != nil {
		return fmt.Errorf("invalid QSTASH_BASE_URL: %w", err)
	}
	targetBaseURL, err := validateHTTPBaseURL(p.targetBaseURL)
	if err != nil {
		return fmt.Errorf("invalid QSTASH_TARGET_BASE_URL: %w", err)
	}

	targetURL := targetBaseURL + path
	publishURL := baseURL + "/v2/publish/" + targetURL
	bodyPayload := payload
	if bodyPayload == nil {
		bodyPayload = map[string]any{}
	}

	body, err := jsoniter.Marshal(bodyPayload)
	if err != nil {
		return fmt.Errorf("marshal job payload: %w", err)
	}
	bodyText := truncateForLog(string(body), 4096)
	curlPreview := buildQStashCurlPreview(publishURL, path, normalizeDelay(delay), p.retries, deduplicationID, bodyText, p.internalJobToken != "")

	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("qstash.publish_url", publishURL),
			attribute.String("qstash.target_url", targetURL),
			attribute.String("qstash.path", path),
			attribute.String("qstash.request_body", bodyText),
			attribute.String("qstash.request_curl_preview", curlPreview),
		)
	}
	p.logger.InfoContext(ctx, "qstash publish request", "path", path, "target_url", targetURL, "publish_url", publishURL, "curl_preview", curlPreview)

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
		return fmt.Errorf("publish qstash job target_url=%s publish_url=%s curl=%s: %w", targetURL, publishURL, curlPreview, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf(
			"publish qstash job status=%d target_url=%s publish_url=%s body=%s curl=%s",
			resp.StatusCode,
			targetURL,
			publishURL,
			strings.TrimSpace(string(raw)),
			curlPreview,
		)
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

func validateHTTPBaseURL(raw string) (string, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return "", fmt.Errorf("value is empty")
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return "", fmt.Errorf("parse %q: %w", candidate, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%q uses unsupported scheme=%q; expected http or https", candidate, parsed.Scheme)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("%q has empty host", candidate)
	}

	return strings.TrimRight(candidate, "/"), nil
}

func buildQStashCurlPreview(
	publishURL string,
	path string,
	delay string,
	retries int,
	deduplicationID string,
	body string,
	withForwardToken bool,
) string {
	parts := []string{
		"curl -X POST",
		shellQuote(publishURL),
		"-H", shellQuote("Authorization: Bearer ***"),
		"-H", shellQuote("Content-Type: application/json"),
		"-H", shellQuote("Upstash-Method: POST"),
	}
	if retries > 0 {
		parts = append(parts, "-H", shellQuote(fmt.Sprintf("Upstash-Retries: %d", retries)))
	}
	if strings.TrimSpace(delay) != "" && delay != "0s" {
		parts = append(parts, "-H", shellQuote("Upstash-Delay: "+delay))
	}
	if strings.TrimSpace(deduplicationID) != "" {
		parts = append(parts, "-H", shellQuote("Upstash-Deduplication-Id: "+strings.TrimSpace(deduplicationID)))
	}
	if withForwardToken {
		parts = append(parts, "-H", shellQuote("Upstash-Forward-X-Internal-Job-Token: ***"))
	}
	parts = append(parts, "-d", shellQuote(body))
	parts = append(parts, "#", shellQuote("path="+path))
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func truncateForLog(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "...(truncated)"
}
