package main

import (
	"context"
	"time"
)

type OpsClock struct{ NowFunc func() time.Time }

func newOpsClock() OpsClock { return OpsClock{NowFunc: time.Now} }
func (c OpsClock) Now() time.Time {
	if c.NowFunc == nil {
		return time.Now().UTC()
	}
	return c.NowFunc().UTC()
}
func (c OpsClock) Stamp() string { return c.Now().Format(time.RFC3339Nano) }

// Per-operation budgets. These cap how long a single work-order (ops) request
// may spend in the data layer so a stalled read/write cannot spin the page
// indefinitely; the request context's own cancel/deadline is layered on top by
// opsContext, so client cancellation still wins.
const (
	opsReadTimeout  = 3 * time.Second
	opsWriteTimeout = 3 * time.Second
)
// opsContext derives a per-operation timeout from the request context. It MUST
// preserve the parent (request) context so a client cancellation or an
// upstream deadline propagates into the store; replacing the parent with
// context.Background() — as this function once did — silently drops those
// signals, so cancel and timeout never reach the data layer.
// opsContext derives a per-operation timeout from the request context. It MUST
// preserve the parent (request) context so a client cancellation or an
// upstream deadline propagates into the store; replacing the parent with
// context.Background() — as this function once did — silently drops those
// signals, so cancel and timeout never reach the data layer.
func opsContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return context.WithTimeout(parent, timeout)
}
func opsDeadline(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	_, ok := ctx.Deadline()
	return ok
}
func opsParseStamp(value string) (time.Time, error) { return time.Parse(time.RFC3339Nano, value) }
func opsBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<uint(attempt-1)) * 20 * time.Millisecond
}
func opsDelay(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func opsAge(now time.Time, stamp string) time.Duration {
	parsed, err := opsParseStamp(stamp)
	if err != nil || now.Before(parsed) {
		return 0
	}
	return now.Sub(parsed)
}
