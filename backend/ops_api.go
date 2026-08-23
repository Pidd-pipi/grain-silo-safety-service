package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type opsAPI struct {
	svc *OpsService
}

func newOpsAPI(svc *OpsService) *opsAPI { return &opsAPI{svc: svc} }

func opsSeedRecords() []OpsRecord {
	return []OpsRecord{
		{ID: "wo-001", Subject: "West tower vent cleaning", Owner: "lin", Status: OpsStatusActive, Priority: OpsPriorityHigh, Labels: map[string]string{"site": "west", "type": "maintenance"}},
		{ID: "wo-002", Subject: "South tower moisture check", Owner: "chen", Status: OpsStatusQueued, Priority: OpsPriorityNormal, Labels: map[string]string{"site": "south", "type": "inspection"}},
	}
}

func (a *opsAPI) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/ops/records", a.handleRecords)
	mux.HandleFunc("/api/ops/records/", a.handleRecord)
	mux.HandleFunc("/api/ops/snapshot", a.handleSnapshot)
}

type opsCreateInput struct {
	ID       string            `json:"id"`
	Subject  string            `json:"subject"`
	Owner    string            `json:"owner"`
	Priority OpsPriority       `json:"priority"`
	Labels   map[string]string `json:"labels"`
}

type opsTransitionInput struct {
	Expected int       `json:"expected"`
	Target   OpsStatus `json:"target"`
	Actor    string    `json:"actor"`
}

func (a *opsAPI) handleRecords(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		page, err := a.svc.Search(r.Context(), opsQueryFromRequest(r))
		if err != nil {
			opsWriteError(w, err)
			return
		}
		opsJSON(w, http.StatusOK, page)
	case http.MethodPost:
		var input opsCreateInput
		if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&input); err != nil {
			opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		recordID := input.ID
		if recordID == "" {
			recordID = newOpsRecordID()
		}
		created, err := a.svc.Create(r.Context(), OpsRecord{ID: recordID, Subject: input.Subject, Owner: input.Owner, Priority: input.Priority, Labels: input.Labels})
		if err != nil {
			opsWriteError(w, err)
			return
		}
		opsJSON(w, http.StatusCreated, created)
	default:
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func opsQueryFromRequest(r *http.Request) OpsQuery {
	q := OpsQuery{
		Subject:  r.URL.Query().Get("subject"),
		Status:   OpsStatus(r.URL.Query().Get("status")),
		Priority: OpsPriority(r.URL.Query().Get("priority")),
		Owner:    r.URL.Query().Get("owner"),
	}
	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
		q.Page = page
	}
	if size, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil {
		q.PageSize = size
	}
	return q
}

func (a *opsAPI) handleRecord(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/transition") {
		a.handleTransition(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/audit") {
		a.handleAudit(w, r)
		return
	}
	if r.Method != http.MethodGet {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := opsPathID(r.URL.Path, "/api/ops/records")
	record, err := a.svc.Get(r.Context(), id)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, record)
}

func (a *opsAPI) handleTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var input opsTransitionInput
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&input); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	actor := input.Actor
	if actor == "" {
		actor = opsActorFromRequest(r)
	}
	id := opsPathID(strings.TrimSuffix(r.URL.Path, "/transition"), "/api/ops/records")
	record, err := a.svc.Transition(r.Context(), id, input.Expected, input.Target, actor)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, record)
}

func (a *opsAPI) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := opsPathID(strings.TrimSuffix(r.URL.Path, "/audit"), "/api/ops/records")
	opsJSON(w, http.StatusOK, map[string]any{"events": a.svc.Audit(id)})
}

func (a *opsAPI) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	opsJSON(w, http.StatusOK, a.svc.Snapshot())
}
