package health

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthReflectsReadiness(t *testing.T) {
	handler := Handler(func() error { return errors.New("store is closed") })
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("health returned 200 when store is down")
	}
}

func TestHealthNotFalseOk(t *testing.T) {
	handler := Handler(func() error { return errors.New("store is closed") })
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("health returned 200 when store is down")
	}
	if strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("health body claims ok when store is down: %s", rec.Body.String())
	}
}
