package health

import "net/http"

func Handler(ready func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"grain-silo-safety-service"}`))
	}
}
