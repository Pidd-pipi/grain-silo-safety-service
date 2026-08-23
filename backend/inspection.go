package main

import (
	"errors"
	"example.com/grain-silo-safety-service/domain"
	"example.com/grain-silo-safety-service/store"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

var inspectionSequence uint64

func newInspectionID() string {
	return fmt.Sprintf("ins-%06d", atomic.AddUint64(&inspectionSequence, 1))
}

type InspectionStatus string

const (
	InspectionStatusPending  InspectionStatus = "pending"
	InspectionStatusReviewed InspectionStatus = "reviewed"
	InspectionStatusClosed   InspectionStatus = "closed"
)

type InspectionRecord struct {
	ID        string           `json:"id"`
	SiloID    string           `json:"siloId"`
	Finding   string           `json:"finding"`
	Severity  OpsPriority      `json:"severity"`
	Status    InspectionStatus `json:"status"`
	Actor     string           `json:"actor"`
	Revision  int              `json:"revision"`
	CreatedAt string           `json:"createdAt"`
	UpdatedAt string           `json:"updatedAt"`
}

type InspectionQuery struct {
	SiloID   string
	Status   InspectionStatus
	Severity OpsPriority
	Page     int
	PageSize int
}

type InspectionPage struct {
	Items    []InspectionRecord `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	Total    int                `json:"total"`
	HasNext  bool               `json:"hasNext"`
}

type inspectionStore struct {
	mu    sync.RWMutex
	items map[string]*InspectionRecord
}

func newInspectionStore() *inspectionStore {
	return &inspectionStore{items: map[string]*InspectionRecord{}}
}

func (s *inspectionStore) Put(rec *InspectionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[rec.ID]; ok {
		return ErrOpsConflict
	}
	copy := *rec
	s.items[rec.ID] = &copy
	return nil
}

func (s *inspectionStore) Get(id string) (*InspectionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.items[id]
	if !ok {
		return nil, ErrOpsNotFound
	}
	copy := *rec
	return &copy, nil
}

func (s *inspectionStore) Update(rec *InspectionRecord, expected int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.items[rec.ID]
	if !ok {
		return ErrOpsNotFound
	}
	if expected > 0 && current.Revision != expected {
		return ErrOpsConflict
	}
	rec.Revision = current.Revision + 1
	rec.UpdatedAt = timeNowOps()
	copy := *rec
	s.items[rec.ID] = &copy
	return nil
}

func (s *inspectionStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrOpsNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *inspectionStore) List() []InspectionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]InspectionRecord, 0, len(s.items))
	for _, rec := range s.items {
		out = append(out, *rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

type InspectionService struct {
	store      *inspectionStore
	silos      *store.Store
	audit      *OpsAudit
	clock      OpsClock
	tokens     chan struct{}
	inFlight   map[string]struct{}
	inFlightMu sync.Mutex
}

func newInspectionService(silos *store.Store, capacity int) *InspectionService {
	if capacity < 1 {
		capacity = 2
	}
	return &InspectionService{
		store:    newInspectionStore(),
		silos:    silos,
		audit:    newOpsAudit(),
		clock:    newOpsClock(),
		tokens:   make(chan struct{}, capacity),
		inFlight: map[string]struct{}{},
	}
}

func classifyFinding(finding string) OpsPriority {
	lower := strings.ToLower(finding)
	switch {
	case strings.Contains(lower, "crack"), strings.Contains(lower, "leak"), strings.Contains(lower, "structural"):
		return OpsPriorityCritical
	case strings.Contains(lower, "dust"), strings.Contains(lower, "vent"), strings.Contains(lower, "corrosion"):
		return OpsPriorityHigh
	default:
		return OpsPriorityNormal
	}
}

var (
	ErrInspectionBusy     = errors.New("inspection capacity exceeded")
	ErrInspectionInFlight = errors.New("inspection already in progress for this silo")
	ErrInspectionState    = errors.New("inspection status transition is not allowed")
)

func (s *InspectionService) Record(siloID, finding, actor string) (*InspectionRecord, error) {
	if !s.acquireToken() {
		return nil, ErrInspectionBusy
	}
	defer s.releaseToken()
	if !s.beginSilo(siloID) {
		return nil, ErrInspectionInFlight
	}
	defer s.endSilo(siloID)

	silo, err := s.silos.Get(siloID)
	if err != nil {
		return nil, err
	}
	if silo.SafetyState == "clear" {
		return nil, domain.ErrSiloRejected
	}
	if finding == "" {
		return nil, domain.ErrFindingEmpty
	}
	rec := &InspectionRecord{
		ID:        newInspectionID(),
		SiloID:    siloID,
		Finding:   finding,
		Severity:  classifyFinding(finding),
		Status:    InspectionStatusPending,
		Actor:     actor,
		Revision:  1,
		CreatedAt: s.clock.Stamp(),
		UpdatedAt: s.clock.Stamp(),
	}
	if err := s.store.Put(rec); err != nil {
		return nil, wrapOps("record", "inspection.put", err)
	}
	if err := s.silos.MarkInspected(siloID, finding); err != nil {
		_ = s.store.Delete(rec.ID)
		return nil, wrapOps("record", "silo.update", err)
	}
	s.audit.Add(rec.ID, "inspection_created", actor)
	return rec, nil
}

func (s *InspectionService) acquireToken() bool {
	select {
	case s.tokens <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *InspectionService) releaseToken() {
	<-s.tokens
}

func (s *InspectionService) beginSilo(siloID string) bool {
	s.inFlightMu.Lock()
	defer s.inFlightMu.Unlock()
	if _, ok := s.inFlight[siloID]; ok {
		return false
	}
	s.inFlight[siloID] = struct{}{}
	return true
}

func (s *InspectionService) endSilo(siloID string) {
	s.inFlightMu.Lock()
	defer s.inFlightMu.Unlock()
	delete(s.inFlight, siloID)
}

func (s *InspectionService) Review(id, actor string) (*InspectionRecord, error) {
	rec, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	if rec.Status != InspectionStatusPending {
		return nil, ErrInspectionState
	}
	rec.Status = InspectionStatusReviewed
	if err := s.store.Update(rec, rec.Revision); err != nil {
		return nil, err
	}
	s.audit.Add(rec.ID, "inspection_reviewed", actor)
	return rec, nil
}

func (s *InspectionService) Close(id, actor string) (*InspectionRecord, error) {
	rec, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	if rec.Status != InspectionStatusReviewed {
		return nil, ErrInspectionState
	}
	rec.Status = InspectionStatusClosed
	if err := s.store.Update(rec, rec.Revision); err != nil {
		return nil, err
	}
	s.audit.Add(rec.ID, "inspection_closed", actor)
	return rec, nil
}

func (s *InspectionService) List(q InspectionQuery) InspectionPage {
	items := s.store.List()
	filtered := make([]InspectionRecord, 0, len(items))
	for _, item := range items {
		if q.SiloID != "" && item.SiloID != q.SiloID {
			continue
		}
		if q.Status != "" && item.Status != q.Status {
			continue
		}
		if q.Severity != "" && item.Severity != q.Severity {
			continue
		}
		filtered = append(filtered, item)
	}
	q = inspectionQueryDefaults(q)
	start, end := inspectionBounds(len(filtered), q.Page, q.PageSize)
	return InspectionPage{Items: filtered[start:end], Page: q.Page, PageSize: q.PageSize, Total: len(filtered), HasNext: end < len(filtered)}
}

func inspectionQueryDefaults(q InspectionQuery) InspectionQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 25
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	return q
}

func inspectionBounds(total, page, size int) (int, int) {
	q := inspectionQueryDefaults(InspectionQuery{Page: page, PageSize: size})
	start := (q.Page - 1) * q.PageSize
	if start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}
	return start, end
}

func (s *InspectionService) ForSilo(siloID string) []InspectionRecord {
	items := s.store.List()
	out := make([]InspectionRecord, 0, len(items))
	for _, item := range items {
		if item.SiloID == siloID {
			out = append(out, item)
		}
	}
	return out
}
