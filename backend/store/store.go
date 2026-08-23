package store

import (
	"example.com/grain-silo-safety-service/domain"
	"fmt"
	"sync"
)

type Store struct {
	mu     sync.RWMutex
	silos  map[string]*domain.Silo
	closed bool
}

func New() *Store {
	notes := make([]string, 1, 4)
	notes[0] = "initial dust sweep"
	items := []*domain.Silo{
		{ID: "silo-01", Name: "West Tower", Grain: "wheat", TemperatureC: 31.4, MoisturePct: 13.1, SafetyState: "review", Notes: notes},
		{ID: "silo-02", Name: "South Tower", Grain: "corn", TemperatureC: 24.2, MoisturePct: 11.8, SafetyState: "clear"},
	}
	silos := make(map[string]*domain.Silo, len(items))
	for _, item := range items {
		silos[item.ID] = item
	}
	return &Store{silos: silos}
}

func (s *Store) List() []domain.Silo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Silo, 0, len(s.silos))
	for _, item := range s.silos {
		// Return a value copy with an isolated Notes slice so callers cannot
		// observe later mutations through a shared backing array.
		result = append(result, cloneSilo(*item))
	}
	return result
}

func (s *Store) Get(id string) (domain.Silo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.silos[id]
	if !ok {
		return domain.Silo{}, fmt.Errorf("%w: %s", domain.ErrSiloNotFound, id)
	}
	// Return a value copy with an isolated Notes slice so callers cannot
	// observe later mutations through a shared backing array.
	return cloneSilo(*item), nil
}

// cloneSilo returns a deep enough copy of the silo that its Notes slice no
// longer shares its backing array with the stored original. This keeps a
// previously returned snapshot stable when the store is mutated later.
func cloneSilo(silo domain.Silo) domain.Silo {
	if silo.Notes == nil {
		return silo
	}
	notes := make([]string, len(silo.Notes))
	copy(notes, silo.Notes)
	silo.Notes = notes
	return silo
}

func (s *Store) Inspect(id, finding string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.silos[id]
	if !ok {
		return fmt.Errorf("%w: %s", domain.ErrSiloNotFound, id)
	}
	if err := domain.RecordInspection(item, finding); err != nil {
		return err
	}
	// Accumulate the finding as a new note rather than overwriting the slice,
	// so the inspection history stays complete.
	item.Notes = appendNote(item.Notes, finding)
	return nil
}

func (s *Store) MarkInspected(id, finding string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.silos[id]
	if !ok {
		return fmt.Errorf("%w: %s", domain.ErrSiloNotFound, id)
	}
	item.LastInspection = finding
	item.Inspected = true
	// Accumulate the finding as a new note rather than overwriting the slice,
	// so the inspection history stays complete.
	item.Notes = appendNote(item.Notes, finding)
	return nil
}

// appendNote appends a note to the slice without aliasing the caller's backing
// array. It is used when mutating a stored silo so that the new note is only
// visible to fresh reads and never mutates a previously returned snapshot.
func appendNote(notes []string, finding string) []string {
	out := make([]string, 0, len(notes)+1)
	out = append(out, notes...)
	out = append(out, finding)
	return out
}

func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

func (s *Store) Ready() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return fmt.Errorf("store is closed")
	}
	return nil
}
