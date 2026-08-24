package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoveryMiddlewareWrites500OnPanic(t *testing.T) {
	server := newEnterpriseServer(":0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/silos", nil)
	rec := httptest.NewRecorder()
	server.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic handler returned %d want 500, body=%q", rec.Code, rec.Body.String())
	}
}
