package store

import (
	"example.com/grain-silo-safety-service/domain"
	"testing"
)

func siloByID(t *testing.T, list []domain.Silo, id string) *domain.Silo {
	t.Helper()
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	t.Fatalf("silo %s not in list: %v", id, list)
	return nil
}

func TestListSnapshotNotRewrittenByInspect(t *testing.T) {
	s := New()
	snapshot := s.List()
	silo := siloByID(t, snapshot, "silo-01")
	if len(silo.Notes) == 0 {
		t.Fatal("seed snapshot missing notes")
	}
	if err := s.Inspect("silo-01", "crack found"); err != nil {
		t.Fatal(err)
	}
	if silo.Notes[0] != "initial dust sweep" {
		t.Fatalf("old List snapshot was rewritten by later Inspect: %v", silo.Notes)
	}
}

func TestListSnapshotNotRewrittenByFormalInspection(t *testing.T) {
	s := New()
	snapshot := s.List()
	silo := siloByID(t, snapshot, "silo-01")
	if err := s.MarkInspected("silo-01", "vent dust high"); err != nil {
		t.Fatal(err)
	}
	if silo.Notes[0] != "initial dust sweep" {
		t.Fatalf("old List snapshot was rewritten by MarkInspected: %v", silo.Notes)
	}
}

func TestGetCopyIsolatedFromStore(t *testing.T) {
	s := New()
	silo, err := s.Get("silo-01")
	if err != nil {
		t.Fatal(err)
	}
	silo.Notes[0] = "tampered"
	again, err := s.Get("silo-01")
	if err != nil {
		t.Fatal(err)
	}
	if again.Notes[0] != "initial dust sweep" {
		t.Fatalf("Get copy is not independent, store polluted: %v", again.Notes)
	}
}

func TestInspectAccumulatesNotes(t *testing.T) {
	s := New()
	if err := s.Inspect("silo-01", "first finding"); err != nil {
		t.Fatal(err)
	}
	if err := s.Inspect("silo-01", "second finding"); err != nil {
		t.Fatal(err)
	}
	list := s.List()
	silo := siloByID(t, list, "silo-01")
	if len(silo.Notes) < 3 {
		t.Fatalf("inspection notes were overwritten instead of accumulated: %v", silo.Notes)
	}
}

func TestFormalInspectionAccumulatesNotes(t *testing.T) {
	s := New()
	if err := s.MarkInspected("silo-01", "first finding"); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkInspected("silo-01", "second finding"); err != nil {
		t.Fatal(err)
	}
	list := s.List()
	silo := siloByID(t, list, "silo-01")
	if len(silo.Notes) < 3 {
		t.Fatalf("formal inspection notes were overwritten instead of accumulated: %v", silo.Notes)
	}
}

func TestListCopyIsolatedFromStore(t *testing.T) {
	s := New()
	list := s.List()
	silo := siloByID(t, list, "silo-01")
	if len(silo.Notes) == 0 {
		t.Fatal("seed list missing notes")
	}
	silo.Notes[0] = "tampered"
	again := s.List()
	after := siloByID(t, again, "silo-01")
	if after.Notes[0] != "initial dust sweep" {
		t.Fatalf("mutating a List result polluted the store: %v", after.Notes)
	}
}
