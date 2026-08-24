package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTransitionConflictDoesNotPolluteHistory(t *testing.T) {
	svc := newOpsService(opsSeedRecords())
	_, err := svc.Transition(context.Background(), "wo-001", 99, OpsStatusPaused, "zhou")
	if !errors.Is(err, ErrOpsConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if history := svc.state.History(); len(history) != 0 {
		t.Fatalf("failed transition polluted state history: %+v", history)
	}
}

func TestIllegalTransitionReturns409(t *testing.T) {
	api := newOpsAPI(newOpsService(opsSeedRecords()))
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records/wo-002/transition", strings.NewReader(`{"target":"paused"}`))
	rec := httptest.NewRecorder()
	api.handleTransition(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("illegal transition status=%d want 409, body=%s", rec.Code, rec.Body.String())
	}
}

func TestTransitionConflictReturns409(t *testing.T) {
	api := newOpsAPI(newOpsService(opsSeedRecords()))
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records/wo-001/transition", strings.NewReader(`{"expected":99,"target":"paused"}`))
	rec := httptest.NewRecorder()
	api.handleTransition(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("conflict transition status=%d want 409, body=%s", rec.Code, rec.Body.String())
	}
}

func TestTransitionResponseHasUpdatedRevision(t *testing.T) {
	api := newOpsAPI(newOpsService(opsSeedRecords()))
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records/wo-001/transition", strings.NewReader(`{"expected":1,"target":"paused"}`))
	rec := httptest.NewRecorder()
	api.handleTransition(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("transition status=%d, body=%s", rec.Code, rec.Body.String())
	}
	var record OpsRecord
	if err := json.Unmarshal(rec.Body.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.Revision != 2 {
		t.Fatalf("transition response revision=%d want 2 (stale revision returned)", record.Revision)
	}
}
