package crawler

import (
	"context"
	"time"
)

// Limiter provides a simple fixed-interval gate.
type Limiter struct {
	ticker *time.Ticker
	stopCh chan struct{}
}

// NewLimiter allows one request every interval.
func NewLimiter(interval time.Duration) *Limiter {
	if interval <= 0 {
		interval = time.Millisecond
	}
	return &Limiter{
		ticker: time.NewTicker(interval),
		stopCh: make(chan struct{}),
	}
}

// Wait blocks until one token becomes available.
func (l *Limiter) Wait(ctx context.Context) error {
	if l == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-l.stopCh:
		return context.Canceled
	case <-l.ticker.C:
		return nil
	}
}

// Stop releases resources used by the limiter.
func (l *Limiter) Stop() {
	if l == nil {
		return
	}
	select {
	case <-l.stopCh:
		return
	default:
		close(l.stopCh)
		l.ticker.Stop()
	}
}
