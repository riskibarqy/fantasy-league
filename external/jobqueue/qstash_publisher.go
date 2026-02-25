package jobqueue

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	sonic "github.com/bytedance/sonic"
	crerr "github.com/cockroachdb/errors"
	"github.com/riskibarqy/fantasy-league/internal/platform/resilience"
	"github.com/valyala/bytebufferpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var errQStashTransient = crerr.New("qstash transient failure")

type QStashPublisherConfig struct {
	BaseURL          string
	Token            string
	TargetBaseURL    string
	Retries          int
	InternalJobToken string
	Timeout          time.Duration
	CircuitBreaker   resilience.CircuitBreakerConfig
}

type QStashPublisher struct {
	client           *http.Client
	baseURL          string
	token            string
	targetBaseURL    string
	retries          int
	internalJobToken string
	logger           *slog.Logger
	breaker          *resilience.CircuitBreaker
	circuitEnabled   bool
}

func NewQStashPublisher(cfg QStashPublisherConfig, logger *slog.Logger) *QStashPublisher {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	breakerCfg := resilience.NormalizeCircuitBreakerConfig(cfg.CircuitBreaker)

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
		breaker:          resilience.NewCircuitBreaker(breakerCfg.FailureThreshold, breakerCfg.OpenTimeout, breakerCfg.HalfOpenMaxReq),
		circuitEnabled:   breakerCfg.Enabled,
	}
}

func (p *QStashPublisher) Enqueue(ctx context.Context, path string, payload any, delay time.Duration, deduplicationID string) error {
	if p.circuitEnabled {
		if err := p.breaker.Allow(); err != nil {
			p.logger.WarnContext(ctx, "qstash circuit breaker rejected request", "state", p.breaker.State())
			return fmt.Errorf("qstash is temporarily unavailable: %w", err)
		}
	}

	path = "/" + strings.TrimLeft(strings.TrimSpace(path), "/")
	if strings.TrimSpace(path) == "/" {
		return crerr.New("job path is required")
	}

	baseURL, err := validateHTTPBaseURL(p.baseURL)
	if err != nil {
		return crerr.Wrap(err, "invalid QSTASH_BASE_URL")
	}
	targetBaseURL, err := validateHTTPBaseURL(p.targetBaseURL)
	if err != nil {
		return crerr.Wrap(err, "invalid QSTASH_TARGET_BASE_URL")
	}

	targetURL := targetBaseURL + path
	publishURL := baseURL + "/v2/publish/" + targetURL
	bodyPayload := payload
	if bodyPayload == nil {
		bodyPayload = map[string]any{}
	}

	body, err := sonic.Marshal(bodyPayload)
	if err != nil {
		return crerr.Wrap(err, "marshal job payload")
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
		return crerr.Wrap(err, "create qstash request")
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
		callErr := fmt.Errorf("%w: publish qstash job target_url=%s publish_url=%s: %v", errQStashTransient, targetURL, publishURL, err)
		p.recordCircuitResult(callErr)
		return callErr
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if isQStashRetryableStatus(resp.StatusCode) {
			callErr := fmt.Errorf(
				"%w: publish qstash job status=%d target_url=%s publish_url=%s body=%s",
				errQStashTransient,
				resp.StatusCode,
				targetURL,
				publishURL,
				strings.TrimSpace(string(raw)),
			)
			p.recordCircuitResult(callErr)
			return callErr
		}

		callErr := fmt.Errorf(
			"publish qstash job status=%d target_url=%s publish_url=%s body=%s",
			resp.StatusCode,
			targetURL,
			publishURL,
			strings.TrimSpace(string(raw)),
		)
		p.recordCircuitResult(callErr)
		return callErr
	}

	p.logger.InfoContext(ctx, "qstash job published", "path", path, "delay", normalizeDelay(delay), "deduplication_id", deduplicationID)
	p.recordCircuitResult(nil)
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
		return "", crerr.New("value is empty")
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return "", crerr.Wrapf(err, "parse %q", candidate)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", crerr.Newf("%q uses unsupported scheme=%q; expected http or https", candidate, parsed.Scheme)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", crerr.Newf("%q has empty host", candidate)
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
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	appendPart := func(part string) {
		if buf.Len() > 0 {
			_ = buf.WriteByte(' ')
		}
		_, _ = buf.WriteString(part)
	}
	appendFlagHeader := func(value string) {
		appendPart("-H")
		appendPart(shellQuote(value))
	}

	appendPart("curl")
	appendPart("-X")
	appendPart("POST")
	appendPart(shellQuote(publishURL))
	appendFlagHeader("Authorization: Bearer ***")
	appendFlagHeader("Content-Type: application/json")
	appendFlagHeader("Upstash-Method: POST")
	if retries > 0 {
		appendFlagHeader("Upstash-Retries: " + strconv.Itoa(retries))
	}
	if strings.TrimSpace(delay) != "" && delay != "0s" {
		appendFlagHeader("Upstash-Delay: " + delay)
	}
	if strings.TrimSpace(deduplicationID) != "" {
		appendFlagHeader("Upstash-Deduplication-Id: " + strings.TrimSpace(deduplicationID))
	}
	if withForwardToken {
		appendFlagHeader("Upstash-Forward-X-Internal-Job-Token: ***")
	}
	appendPart("-d")
	appendPart(shellQuote(body))
	appendPart("#")
	appendPart(shellQuote("path=" + path))

	return buf.String()
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

func (p *QStashPublisher) recordCircuitResult(err error) {
	if !p.circuitEnabled || p.breaker == nil {
		return
	}
	if err == nil {
		p.breaker.RecordSuccess()
		return
	}
	if isQStashCircuitFailure(err) {
		p.breaker.RecordFailure()
		return
	}
	p.breaker.RecordSuccess()
}

func isQStashCircuitFailure(err error) bool {
	if err == nil {
		return false
	}
	return stderrors.Is(err, errQStashTransient)
}

func isQStashRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}
