package main

import (
	"encoding/json"
	"example.com/grain-silo-safety-service/api"
	"example.com/grain-silo-safety-service/store"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

func TestAlertStoreConcurrentEvaluateAndReadNoRace(t *testing.T) {
	alertStore := newAlertStore()
	svc := newAlertService(alertStore, store.New(), newOpsClock())
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			svc.EvaluateAll()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			_ = alertStore.ListEvents()
			_ = alertStore.ListRules()
			if i%7 == 0 {
				_, _ = alertStore.CreateRule(AlertRule{ID: fmt.Sprintf("r-%d", i), Metric: "temperature", Max: 40, Enabled: true})
			}
		}
	}()
	close(start)
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("concurrent test timed out")
	}
}

func TestAlertEventsSnapshotStable(t *testing.T) {
	alertStore := newAlertStore()
	svc := newAlertService(alertStore, store.New(), newOpsClock())
	svc.EvaluateAll()
	if alertStore.CountEvents() == 0 {
		t.Fatal("expected seeded rules to emit events")
	}
	snapshot := alertStore.ListEvents()
	if len(snapshot) == 0 {
		t.Fatal("snapshot unexpectedly empty")
	}
	for i := range snapshot {
		snapshot[i].Level = "tampered"
	}
	svc.EvaluateAll()
	for _, ev := range alertStore.ListEvents() {
		if ev.Level == "tampered" {
			t.Fatalf("store events polluted by caller mutation: %+v", ev)
		}
	}
}

func TestAlertServiceEventsSnapshotStable(t *testing.T) {
	alertStore := newAlertStore()
	svc := newAlertService(alertStore, store.New(), newOpsClock())
	svc.EvaluateAll()
	snapshot := svc.Events()
	if len(snapshot) == 0 {
		t.Fatal("service events snapshot unexpectedly empty")
	}
	for i := range snapshot {
		snapshot[i].Value = 999
	}
	for _, ev := range svc.Events() {
		if ev.Value == 999 {
			t.Fatalf("service events polluted by caller mutation: %+v", ev)
		}
	}
}

func TestAlertEventsEndpointConcurrentNoRace(t *testing.T) {
	alertStore := newAlertStore()
	svc := newAlertService(alertStore, store.New(), newOpsClock())
	alerts := newAlertsAPI(svc)
	webFS := fs.FS(fstest.MapFS{"index.html": {Data: []byte("silos")}, "app.js": {Data: []byte("x")}})
	handler := api.NewRouter(store.New(), webFS, alerts)
	server := httptest.NewServer(handler)
	defer server.Close()
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			svc.EvaluateAll()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			resp, err := http.Get(server.URL + "/api/alerts/events")
			if err != nil {
				t.Errorf("GET failed: %v", err)
				return
			}
			var body struct {
				Events []AlertEvent `json:"events"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
		}
	}()
	close(start)
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("concurrent endpoint test timed out")
	}
}

func TestAlertEventCountConcurrentNoRace(t *testing.T) {
	alertStore := newAlertStore()
	svc := newAlertService(alertStore, store.New(), newOpsClock())
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			svc.EvaluateAll()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			_ = svc.EventCount()
		}
	}()
	close(start)
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("concurrent count test timed out")
	}
}
