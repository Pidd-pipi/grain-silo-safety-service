package api

import (
	"encoding/json"
	"errors"
	"example.com/grain-silo-safety-service/domain"
	"example.com/grain-silo-safety-service/store"
	"example.com/grain-silo-safety-service/validation"
	"net/http"
	"strings"
)

func listSilos(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, 405, "method not allowed")
			return
		}
		writeJSON(w, 200, map[string]any{"silos": s.List()})
	}
}
func inspectSilo(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/inspect") {
			writeError(w, 404, "endpoint not found")
			return
		}
		input, err := validation.DecodeInspection(r)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		if err = s.Inspect(siloID(r.URL.Path), input.Finding); err != nil {
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, domain.ErrSiloNotFound):
				status = 404
			case errors.Is(err, domain.ErrSiloRejected):
				status = 409
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, 200, map[string]string{"status": "inspection-recorded", "siloID": siloID(r.URL.Path)})
	}
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
