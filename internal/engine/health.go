package engine

import (
	"context"
	"net/http"
	"time"
)

// CheckHealth performs an HTTP GET against the engine's health endpoint and
// reports whether the engine responded with a 2xx status. The timeout bounds
// the whole request; a default of 2s is used when timeout <= 0.
func CheckHealth(def *EngineDef, timeout time.Duration) bool {
	if def.BaseURL == "" {
		return false
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, def.HealthURL(), nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
