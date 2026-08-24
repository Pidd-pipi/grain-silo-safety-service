package health

import (
	"encoding/json"
	"net/http"
)

func Handler(ready func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := ready(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "error",
				"service": "grain-silo-safety-service",
				"error":   err.Error(),
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "grain-silo-safety-service",
		})
	}
}
