package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpsContextPropagatesParentCancel(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	ctx, _ := opsContext(parent, 3*time.Second)
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("opsContext did not propagate parent cancellation: %v", ctx.Err())
	}
}

func TestOpsStoreGetHonorsCanceledContext(t *testing.T) {
	store := newOpsStore(opsSeedRecords())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.Get(ctx, "wo-001")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Get with canceled ctx returned %v, want context.Canceled", err)
	}
}

func TestOpsStoreListHonorsCanceledContext(t *testing.T) {
	store := newOpsStore(opsSeedRecords())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.List(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("List with canceled ctx returned %v, want context.Canceled", err)
	}
}

func TestOpsStorePutHonorsCanceledContext(t *testing.T) {
	store := newOpsStore(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := store.Put(ctx, OpsRecord{ID: "wo-900", Subject: "x", Owner: "o", Priority: OpsPriorityLow, Labels: map[string]string{"site": "west"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Put with canceled ctx returned %v, want context.Canceled", err)
	}
}

func TestOpsAPIUsesRequestContext(t *testing.T) {
	api := newOpsAPI(newOpsService(opsSeedRecords()))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/ops/records", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	api.handleRecords(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("canceled request still returned 200, body=%s", rec.Body.String())
	}
}

func TestOpsAPIGetUsesRequestContext(t *testing.T) {
	api := newOpsAPI(newOpsService(opsSeedRecords()))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/ops/records/wo-001", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	api.handleRecord(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("canceled request still returned 200, body=%s", rec.Body.String())
	}
}
