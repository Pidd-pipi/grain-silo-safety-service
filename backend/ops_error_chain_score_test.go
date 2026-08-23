package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpsErrorChainPreserved(t *testing.T) {
	err := wrapOps("get", "store.get", ErrOpsNotFound)
	if !errors.Is(err, ErrOpsNotFound) {
		t.Fatalf("wrapOps broke errors.Is chain: %v", err)
	}
	var typed *OpsError
	if !errors.As(err, &typed) {
		t.Fatalf("wrapOps error is not an *OpsError: %v", err)
	}
	if typed.Cause == nil {
		t.Fatalf("wrapOps lost the wrapped cause")
	}
}

func TestOpsRecordNotFoundReturns404(t *testing.T) {
	api := newOpsAPI(newOpsService(opsSeedRecords()))
	req := httptest.NewRequest(http.MethodGet, "/api/ops/records/nope", nil)
	rec := httptest.NewRecorder()
	api.handleRecord(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404, body=%s", rec.Code, rec.Body.String())
	}
}

func TestOpsConflictStillMapsTo409(t *testing.T) {
	api := newOpsAPI(newOpsService(opsSeedRecords()))
	body := `{"id":"wo-001","subject":"dup","owner":"o","priority":"high","labels":{"site":"west"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records", strings.NewReader(body))
	rec := httptest.NewRecorder()
	api.handleRecords(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate create status=%d want 409, body=%s", rec.Code, rec.Body.String())
	}
}

func TestOpsErrorUnwrapPresent(t *testing.T) {
	err := wrapOps("get", "store.get", ErrOpsNotFound)
	var typed *OpsError
	if !errors.As(err, &typed) {
		t.Fatalf("not an *OpsError: %v", err)
	}
	if typed.Unwrap() == nil {
		t.Fatalf("OpsError.Unwrap returned nil; errors.Is cannot traverse the chain")
	}
}
