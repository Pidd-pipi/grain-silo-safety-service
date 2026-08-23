package api

import (
	"example.com/grain-silo-safety-service/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func newScoreHandler() http.Handler {
	return NewRouter(store.New(), fstest.MapFS{"index.html": {Data: []byte("silos")}, "app.js": {Data: []byte("x")}})
}

func TestInspectClearSiloReturns409(t *testing.T) {
	server := httptest.NewServer(newScoreHandler())
	defer server.Close()
	resp, err := http.Post(server.URL+"/api/silos/silo-02/inspect", "application/json", strings.NewReader(`{"finding":"routine check"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("clear silo inspect got %d want 409", resp.StatusCode)
	}
}

func TestInspectUnknownSiloReturns404(t *testing.T) {
	server := httptest.NewServer(newScoreHandler())
	defer server.Close()
	resp, err := http.Post(server.URL+"/api/silos/nope/inspect", "application/json", strings.NewReader(`{"finding":"routine check"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown silo inspect got %d want 404", resp.StatusCode)
	}
}
