package main

import (
	"context"
	"example.com/grain-silo-safety-service/store"
	"testing"
	"time"
)

func TestSweeperStopsOnCancel(t *testing.T) {
	svc := newAlertService(newAlertStore(), store.New(), newOpsClock())
	sw := newAlertSweeper(svc, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	go func() {
		close(started)
		sw.Run(ctx)
	}()
	<-started
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-sw.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("sweeper goroutine did not exit after context cancel (goroutine leak)")
	}
}

func TestSweeperStopHaltsWorker(t *testing.T) {
	svc := newAlertService(newAlertStore(), store.New(), newOpsClock())
	sw := newAlertSweeper(svc, time.Millisecond)
	go sw.Run(context.Background())
	time.Sleep(20 * time.Millisecond)
	sw.Stop()
	select {
	case <-sw.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("sweeper goroutine did not exit after Stop (goroutine leak)")
	}
}

func TestSweeperZeroIntervalDefaults(t *testing.T) {
	svc := newAlertService(newAlertStore(), store.New(), newOpsClock())
	sw := newAlertSweeper(svc, 0)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sw.Run(ctx)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sweeper with zero interval did not exit")
	}
}

func TestStartSweeperStopsWorkerOnStop(t *testing.T) {
	svc := newAlertService(newAlertStore(), store.New(), newOpsClock())
	sw := newAlertSweeper(svc, time.Millisecond)
	stop := startSweeper(sw)
	time.Sleep(20 * time.Millisecond)
	stop()
	select {
	case <-sw.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("sweeper started via startSweeper did not exit after stop (goroutine leak)")
	}
}
