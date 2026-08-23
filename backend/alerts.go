package main

import (
	"example.com/grain-silo-safety-service/store"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

var alertSequence uint64

func newAlertID() string { return fmt.Sprintf("alert-%06d", atomic.AddUint64(&alertSequence, 1)) }

type AlertRule struct {
	ID      string  `json:"id"`
	Metric  string  `json:"metric"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Level   string  `json:"level"`
	Enabled bool    `json:"enabled"`
}

type AlertEvent struct {
	ID     string  `json:"id"`
	RuleID string  `json:"ruleId"`
	SiloID string  `json:"siloId"`
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	Level  string  `json:"level"`
	At     string  `json:"at"`
}

type AlertStore struct {
	mu     sync.RWMutex
	rules  map[string]*AlertRule
	events []AlertEvent
}

func newAlertStore() *AlertStore {
	return &AlertStore{
		rules: map[string]*AlertRule{
			"temp-high":  {ID: "temp-high", Metric: "temperature", Max: 30, Level: "warning", Enabled: true},
			"moist-high": {ID: "moist-high", Metric: "moisture", Max: 13, Level: "warning", Enabled: true},
		},
		events: []AlertEvent{},
	}
}

func (s *AlertStore) CreateRule(rule AlertRule) (AlertRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rule.ID == "" {
		rule.ID = newAlertID()
	}
	if _, ok := s.rules[rule.ID]; ok {
		return AlertRule{}, ErrOpsConflict
	}
	copy := rule
	s.rules[copy.ID] = &copy
	return copy, nil
}

func (s *AlertStore) ListRules() []AlertRule {
	out := make([]AlertRule, 0, len(s.rules))
	for _, rule := range s.rules {
		out = append(out, *rule)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *AlertStore) AppendEvent(ev AlertEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
}

func (s *AlertStore) ListEvents() []AlertEvent {
	return s.events
}

func (s *AlertStore) CountEvents() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

type AlertService struct {
	store *AlertStore
	silos *store.Store
	clock OpsClock
}

func newAlertService(alertStore *AlertStore, silos *store.Store, clock OpsClock) *AlertService {
	return &AlertService{store: alertStore, silos: silos, clock: clock}
}

func (s *AlertService) CreateRule(rule AlertRule) (AlertRule, error) {
	rule.Metric = strings.ToLower(strings.TrimSpace(rule.Metric))
	if rule.Metric != "temperature" && rule.Metric != "moisture" {
		return AlertRule{}, fmt.Errorf("%w: unknown metric %q", ErrOpsInvalid, rule.Metric)
	}
	if rule.Min > rule.Max {
		return AlertRule{}, fmt.Errorf("%w: min must not exceed max", ErrOpsInvalid)
	}
	return s.store.CreateRule(rule)
}

func (s *AlertService) EvaluateAll() int {
	rules := s.store.ListRules()
	silos := s.silos.List()
	count := 0
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		for _, silo := range silos {
			var value float64
			switch rule.Metric {
			case "temperature":
				value = silo.TemperatureC
			case "moisture":
				value = silo.MoisturePct
			}
			if value < rule.Min || value > rule.Max {
				s.store.AppendEvent(AlertEvent{
					ID:     newAlertID(),
					RuleID: rule.ID,
					SiloID: silo.ID,
					Metric: rule.Metric,
					Value:  value,
					Level:  rule.Level,
					At:     s.clock.Stamp(),
				})
				count++
			}
		}
	}
	return count
}

func (s *AlertService) Rules() []AlertRule   { return s.store.ListRules() }
func (s *AlertService) Events() []AlertEvent { return s.store.events }
func (s *AlertService) EventCount() int      { return len(s.store.events) }
