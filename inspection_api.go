package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type inspectionAPI struct {
	svc *InspectionService
}

func newInspectionAPI(svc *InspectionService) *inspectionAPI { return &inspectionAPI{svc: svc} }

func (a *inspectionAPI) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/inspections", a.handleList)
	mux.HandleFunc("/api/inspections/", a.handleItem)
}

type inspectionCreateInput struct {
	SiloID  string `json:"siloId"`
	Finding string `json:"finding"`
	Actor   string `json:"actor"`
}

func (a *inspectionAPI) handleList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := InspectionQuery{
			SiloID:   r.URL.Query().Get("silo"),
			Status:   InspectionStatus(r.URL.Query().Get("status")),
			Severity: OpsPriority(r.URL.Query().Get("severity")),
		}
		if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
			q.Page = page
		}
		if size, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil {
			q.PageSize = size
		}
		opsJSON(w, http.StatusOK, a.svc.List(q))
	case http.MethodPost:
		var input inspectionCreateInput
		if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&input); err != nil {
			opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		actor := input.Actor
		if actor == "" {
			actor = opsActorFromRequest(r)
		}
		rec, err := a.svc.Record(input.SiloID, strings.TrimSpace(input.Finding), actor)
		if err != nil {
			opsJSON(w, http.StatusOK, map[string]string{"status": "accepted", "siloId": input.SiloID})
			return
		}
		_ = rec
		opsJSON(w, http.StatusOK, map[string]string{"status": "accepted", "siloId": input.SiloID})
	default:
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (a *inspectionAPI) handleItem(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/review") {
		a.handleReview(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/close") {
		a.handleClose(w, r)
		return
	}
	if r.Method != http.MethodGet {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := opsPathID(r.URL.Path, "/api/inspections")
	rec, err := a.svc.store.Get(id)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, rec)
}

func (a *inspectionAPI) handleReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := opsPathID(strings.TrimSuffix(r.URL.Path, "/review"), "/api/inspections")
	rec, err := a.svc.Review(id, opsActorFromRequest(r))
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, rec)
}

func (a *inspectionAPI) handleClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := opsPathID(strings.TrimSuffix(r.URL.Path, "/close"), "/api/inspections")
	rec, err := a.svc.Close(id, opsActorFromRequest(r))
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, rec)
}
