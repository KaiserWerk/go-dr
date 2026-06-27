package crawler

import (
	"context"
	"io"
	"net/http"
)

// HTTPDoer abstracts the http.Client for testability.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a small composable fetcher with optional rate limiting and retries.
type Client struct {
	httpClient HTTPDoer
	userAgent  string
	limiter    *Limiter
	retry      RetryPolicy
}

// Config configures a crawler client.
type Config struct {
	HTTPClient HTTPDoer
	UserAgent  string
	Limiter    *Limiter
	Retry      RetryPolicy
}

// NewClient creates a new crawler client.
func NewClient(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}

	return &Client{
		httpClient: hc,
		userAgent:  cfg.UserAgent,
		limiter:    cfg.Limiter,
		retry:      cfg.Retry,
	}
}

// Get fetches a URL and returns the status code and body bytes.
func (c *Client) Get(ctx context.Context, url string) (int, []byte, error) {
	fetch := func() (int, []byte, error) {
		if c.limiter != nil {
			if err := c.limiter.Wait(ctx); err != nil {
				return 0, nil, err
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return 0, nil, err
		}
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer resp.Body.Close()

		payload, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, nil, err
		}

		return resp.StatusCode, payload, nil
	}

	if c.retry.MaxAttempts <= 1 {
		return fetch()
	}

	return DoWithRetry(ctx, c.retry, fetch)
}
