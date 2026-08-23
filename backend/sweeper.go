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

func (sw *alertSweeper) Run(ctx context.Context) {
	defer close(sw.doneCh)
	if sw.every <= 0 {
		sw.every = time.Minute
	}
	ticker := time.NewTicker(sw.every)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sw.svc.EvaluateAll()
		case <-sw.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (sw *alertSweeper) Stop() { close(sw.stopCh) }

func (sw *alertSweeper) Done() <-chan struct{} { return sw.doneCh }
