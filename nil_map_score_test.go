package main

import (
	"context"
	"errors"
	"example.com/grain-silo-safety-service/api"
	"example.com/grain-silo-safety-service/store"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func apiRouter(webFS fs.FS, registrars ...api.Registrar) http.Handler {
	return api.NewRouter(store.New(), webFS, registrars...)
}

func TestAuditAddInitializesDetails(t *testing.T) {
	audit := newOpsAudit()
	event := audit.Add("rec-1", "created", "zhou")
	if event.Details == nil {
		t.Fatalf("audit event Details is nil")
	}
	if event.Details["by"] != "zhou" {
		t.Fatalf("audit event Details.by=%q want zhou", event.Details["by"])
	}
}

func TestAuditByTypeCounts(t *testing.T) {
	audit := newOpsAudit()
	audit.Add("rec-1", "created", "zhou")
	audit.Add("rec-2", "created", "chen")
	audit.Add("rec-3", "status_changed", "wu")
	if got := audit.ByType("created"); got != 2 {
		t.Fatalf("ByType(created)=%d want 2", got)
	}
}

func TestOpsCreateWithoutLabelsNoPanic(t *testing.T) {
	svc := newOpsService(nil)
	_, err := svc.Create(context.Background(), OpsRecord{ID: "wo-900", Subject: "belt check", Owner: "wu", Priority: OpsPriorityHigh})
	if err == nil {
		t.Fatal("expected create without site label to be rejected by policy")
	}
	if !errors.Is(err, ErrOpsPolicy) {
		t.Fatalf("unexpected error type: %v", err)
	}
}

func TestOpsCloneLabelsIndependent(t *testing.T) {
	ops := newOpsStore(opsSeedRecords())
	got, err := ops.Get(context.Background(), "wo-001")
	if err != nil {
		t.Fatal(err)
	}
	got.Labels["tampered"] = "yes"
	again, err := ops.Get(context.Background(), "wo-001")
	if err != nil {
		t.Fatal(err)
	}
	if again.Labels["tampered"] != "" {
		t.Fatalf("labels leaked across clones: %v", again.Labels)
	}
}

func TestOpsCreateWithoutLabelsHTTP(t *testing.T) {
	svc := newOpsService(nil)
	api := newOpsAPI(svc)
	webFS := fs.FS(fstest.MapFS{"index.html": {Data: []byte("silos")}, "app.js": {Data: []byte("x")}})
	handler := recoveryMiddleware(apiRouter(webFS, api))
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records", strings.NewReader(`{"subject":"belt check","owner":"wu","priority":"high"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want 422, body=%s", rec.Code, rec.Body.String())
	}
}
