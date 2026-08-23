package api

import (
	"example.com/grain-silo-safety-service/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testHandler() http.Handler {
	return NewRouter(store.New(), fstest.MapFS{"index.html": {Data: []byte("silos")}, "app.js": {Data: []byte("fetch('/api/silos')")}})
}
func TestPublicRoutes(t *testing.T) {
	server := httptest.NewServer(testHandler())
	defer server.Close()
	for _, path := range []string{"/healthz", "/api/silos", "/", "/app.js"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("%s: %d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}
func TestInspectionValidation(t *testing.T) {
	server := httptest.NewServer(testHandler())
	defer server.Close()
	bad, _ := http.Post(server.URL+"/api/silos/silo-01/inspect", "application/json", strings.NewReader(`{"finding":""}`))
	if bad.StatusCode != 400 {
		t.Fatalf("got %d", bad.StatusCode)
	}
	bad.Body.Close()
	good, _ := http.Post(server.URL+"/api/silos/silo-01/inspect", "application/json", strings.NewReader(`{"finding":"Clear dust near vent"}`))
	if good.StatusCode != 200 {
		t.Fatalf("got %d", good.StatusCode)
	}
	good.Body.Close()
}
