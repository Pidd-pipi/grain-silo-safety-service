package api

import (
	"example.com/grain-silo-safety-service/store"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestStaticMissingAssetReturns404(t *testing.T) {
	router := NewRouter(store.New(), fstest.MapFS{"index.html": {Data: []byte("silos")}, "app.js": {Data: []byte("x")}})
	req := httptest.NewRequest(http.MethodGet, "/no-such-asset.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing asset returned %d want 404, body=%q", rec.Code, rec.Body.String())
	}
}
