package main

import (
	"context"
	"time"
)

type alertSweeper struct {
	svc    *AlertService
	every  time.Duration
	stopCh chan struct{}
	doneCh chan struct{}
}

func newAlertSweeper(svc *AlertService, every time.Duration) *alertSweeper {
	return &alertSweeper{svc: svc, every: every, stopCh: make(chan struct{}), doneCh: make(chan struct{})}
}

// Run drives the alert evaluation loop until either ctx is cancelled or Stop
// is called, then closes doneCh so callers can wait for the worker to exit.
// Both signals are honoured so the goroutine never outlives a graceful
// shutdown: a parent context cancellation (e.g. the shutdown context wired in
// by main) stops it, and an explicit Stop() stops it too.
func (sw *alertSweeper) Run(ctx context.Context) {
	if sw.every <= 0 {
		sw.every = time.Minute
	}
	ticker := time.NewTicker(sw.every)
	defer ticker.Stop()
	defer close(sw.doneCh)
	for {
		select {
		case <-ticker.C:
			sw.svc.EvaluateAll()
		case <-ctx.Done():
			return
		case <-sw.stopCh:
			return
		}
	}
}

// Stop signals the worker to exit and blocks until Run has actually returned,
// so the caller can be certain the background sweep goroutine is no longer
// running before the process tears down. Safe to call more than once.
func (sw *alertSweeper) Stop() {
	select {
	case <-sw.stopCh:
		// already closed; fall through to wait on doneCh
	default:
		close(sw.stopCh)
	}
	<-sw.doneCh
}

func (sw *alertSweeper) Done() <-chan struct{} { return sw.doneCh }
