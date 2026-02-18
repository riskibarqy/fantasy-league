package httpapi

import (
	"context"
	"net"
	"net/http"
	"strings"
)

func resolveClientIP(ctx context.Context, r *http.Request) string {
	_ = ctx

	candidates := []string{
		r.Header.Get("Fly-Client-IP"),
		r.Header.Get("X-Forwarded-For"),
		r.Header.Get("X-Real-IP"),
		r.RemoteAddr,
	}

	for _, candidate := range candidates {
		if ip := normalizeIP(candidate); ip != "" {
			return ip
		}
	}

	return ""
}

func resolveCountryCode(ctx context.Context, r *http.Request) string {
	_ = ctx

	candidates := []string{
		r.Header.Get("Fly-Client-Country"),
		r.Header.Get("CF-IPCountry"),
		r.Header.Get("X-Vercel-IP-Country"),
		r.Header.Get("X-AppEngine-Country"),
		r.Header.Get("CloudFront-Viewer-Country"),
	}

	for _, candidate := range candidates {
		code := normalizeCountry(candidate)
		if code != "" {
			return code
		}
	}

	return "ZZ"
}

func normalizeIP(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.Contains(value, ",") {
		value = strings.TrimSpace(strings.Split(value, ",")[0])
	}

	if host, _, err := net.SplitHostPort(value); err == nil {
		value = strings.TrimSpace(host)
	}

	parsed := net.ParseIP(value)
	if parsed == nil {
		return ""
	}
	return parsed.String()
}

func normalizeCountry(raw string) string {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) != 2 {
		return ""
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return code
}
