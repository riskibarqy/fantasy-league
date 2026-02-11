package anubis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/domain/user"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

type Client struct {
	httpClient    *http.Client
	introspectURL string
	logger        *slog.Logger
}

func NewClient(httpClient *http.Client, baseURL, introspectPath string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{
		httpClient:    httpClient,
		introspectURL: buildURL(baseURL, introspectPath),
		logger:        logger,
	}
}

func (c *Client) VerifyAccessToken(ctx context.Context, token string) (user.Principal, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return user.Principal{}, fmt.Errorf("%w: token is required", usecase.ErrUnauthorized)
	}

	payload := introspectRequest{Token: token}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return user.Principal{}, fmt.Errorf("marshal introspect request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.introspectURL, bytes.NewReader(encoded))
	if err != nil {
		return user.Principal{}, fmt.Errorf("create introspect request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return user.Principal{}, fmt.Errorf("request introspection to anubis: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return user.Principal{}, fmt.Errorf("%w: introspection denied", usecase.ErrUnauthorized)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return user.Principal{}, fmt.Errorf("read introspect response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.WarnContext(ctx, "anubis introspection non-200",
			"status_code", resp.StatusCode,
		)
		return user.Principal{}, fmt.Errorf("anubis introspection failed with status %d", resp.StatusCode)
	}

	var decoded introspectResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return user.Principal{}, fmt.Errorf("unmarshal introspect response: %w", err)
	}

	if !decoded.Active {
		return user.Principal{}, fmt.Errorf("%w: inactive token", usecase.ErrUnauthorized)
	}
	if strings.TrimSpace(decoded.UserID) == "" {
		return user.Principal{}, fmt.Errorf("invalid introspect response: user_id is empty")
	}

	return user.Principal{
		UserID: decoded.UserID,
		Email:  decoded.Email,
	}, nil
}

type introspectRequest struct {
	Token string `json:"token"`
}

type introspectResponse struct {
	Active bool   `json:"active"`
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

func buildURL(baseURL, path string) string {
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	path = strings.TrimSpace(path)
	if path == "" {
		return baseURL
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return baseURL + path
}
