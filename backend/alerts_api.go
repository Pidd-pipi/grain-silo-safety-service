package main

import (
	"encoding/json"
	"io"
	"net/http"
)

type alertsAPI struct {
	svc *AlertService
}

func newAlertsAPI(svc *AlertService) *alertsAPI { return &alertsAPI{svc: svc} }

func (a *alertsAPI) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/alerts/rules", a.handleRules)
	mux.HandleFunc("/api/alerts/events", a.handleEvents)
}

func (a *alertsAPI) handleRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		opsJSON(w, http.StatusOK, map[string]any{"rules": a.svc.Rules()})
	case http.MethodPost:
		var input AlertRule
		if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&input); err != nil {
			opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		rule, err := a.svc.CreateRule(input)
		if err != nil {
			opsWriteError(w, err)
			return
		}
		opsJSON(w, http.StatusCreated, rule)
	default:
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (a *alertsAPI) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	opsJSON(w, http.StatusOK, map[string]any{"events": a.svc.store.events})
}
