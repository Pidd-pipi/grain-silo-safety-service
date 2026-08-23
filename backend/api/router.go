package api

import (
	"example.com/grain-silo-safety-service/health"
	"example.com/grain-silo-safety-service/store"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Registrar lets extra API groups (work orders, inspections, alerts) attach
// their routes to the shared mux.
type Registrar interface {
	Register(mux *http.ServeMux)
}

func NewRouter(s *store.Store, webFS fs.FS, registrars ...Registrar) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Handler(s.Ready))
	mux.HandleFunc("/api/silos", listSilos(s))
	mux.HandleFunc("/api/silos/", inspectSilo(s))
	for _, registrar := range registrars {
		registrar.Register(mux)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		asset := r.URL.Path
		if asset == "/" {
			asset = "/index.html"
		}
		data, err := fs.ReadFile(webFS, path.Clean(strings.TrimPrefix(asset, "/")))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(asset, ".js") {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		_, _ = w.Write(data)
	})
	return mux
}
func siloID(path string) string {
	return strings.TrimSuffix(strings.TrimPrefix(path, "/api/silos/"), "/inspect")
}
