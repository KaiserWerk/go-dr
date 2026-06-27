package crawler

import (
	"context"
	"errors"
	"time"
)

// RetryPolicy controls retry behavior.
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	ShouldRetry func(statusCode int, err error) bool
}

// DefaultShouldRetry retries on network errors and HTTP 5xx.
func DefaultShouldRetry(statusCode int, err error) bool {
	if err != nil {
		return true
	}
	return statusCode >= 500
}

// DoWithRetry executes fn until it succeeds or attempts are exhausted.
func DoWithRetry(
	ctx context.Context,
	policy RetryPolicy,
	fn func() (int, []byte, error),
) (int, []byte, error) {
	if policy.MaxAttempts < 1 {
		policy.MaxAttempts = 1
	}
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = 250 * time.Millisecond
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 5 * time.Second
	}
	if policy.ShouldRetry == nil {
		policy.ShouldRetry = DefaultShouldRetry
	}

	var lastStatus int
	var lastBody []byte
	var lastErr error

	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		status, body, err := fn()
		if err == nil && !policy.ShouldRetry(status, nil) {
			return status, body, nil
		}
		if err == nil && status < 500 {
			return status, body, nil
		}

		lastStatus = status
		lastBody = body
		lastErr = err

		if attempt == policy.MaxAttempts || !policy.ShouldRetry(status, err) {
			break
		}

		delay := backoff(attempt, policy.BaseDelay, policy.MaxDelay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return lastStatus, lastBody, ctx.Err()
		case <-timer.C:
		}
	}

	if lastErr != nil {
		return lastStatus, lastBody, lastErr
	}
	return lastStatus, lastBody, errors.New("retry attempts exhausted")
}

func backoff(attempt int, base, max time.Duration) time.Duration {
	d := base << (attempt - 1)
	if d > max {
		return max
	}
	return d
}
