package main

import (
	"errors"
	"example.com/grain-silo-safety-service/domain"
	"example.com/grain-silo-safety-service/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInspectionTokenReleasedOnFailure(t *testing.T) {
	svc := newInspectionService(store.New(), 2)
	for i := 0; i < 2; i++ {
		_, err := svc.Record("no-such-silo", "x", "zhou")
		if !errors.Is(err, domain.ErrSiloNotFound) {
			t.Fatalf("attempt %d: unexpected error %v", i, err)
		}
	}
	rec, err := svc.Record("silo-01", "crack found", "zhou")
	if err != nil {
		t.Fatalf("valid inspection after two failed requests was rejected: %v", err)
	}
	if rec == nil || rec.Status != InspectionStatusPending {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestInspectionPerSiloReleasedOnFailure(t *testing.T) {
	svc := newInspectionService(store.New(), 2)
	_, err := svc.Record("silo-01", "", "zhou")
	if !errors.Is(err, domain.ErrFindingEmpty) {
		t.Fatalf("empty finding should fail: %v", err)
	}
	rec, err := svc.Record("silo-01", "crack found", "zhou")
	if err != nil {
		t.Fatalf("valid inspection after a failed one was rejected (per-silo guard leaked): %v", err)
	}
	if rec == nil || rec.Status != InspectionStatusPending {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestInspectionCreateDoesNotSwallowError(t *testing.T) {
	api := newInspectionAPI(newInspectionService(store.New(), 2))
	req := httptest.NewRequest(http.MethodPost, "/api/inspections", strings.NewReader(`{"siloId":"no-such-silo","finding":"check"}`))
	req.Header.Set("X-Operator", "zhou")
	rec := httptest.NewRecorder()
	api.handleList(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404, body=%s", rec.Code, rec.Body.String())
	}
}

func TestInspectionBusyReturns503(t *testing.T) {
	api := newInspectionAPI(newInspectionService(store.New(), 1))
	// exhaust the single token via two failing requests (token leaked in bug env)
	_, _ = api.svc.Record("no-such-silo", "x", "zhou")
	_, _ = api.svc.Record("no-such-silo", "y", "zhou")
	req := httptest.NewRequest(http.MethodPost, "/api/inspections", strings.NewReader(`{"siloId":"silo-01","finding":"check"}`))
	rec := httptest.NewRecorder()
	api.handleList(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("busy inspection must not be silently accepted, got 200 body=%s", rec.Body.String())
	}
}
