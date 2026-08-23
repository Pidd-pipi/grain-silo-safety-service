package domain

import (
	"errors"
	"testing"
)

func TestRecordInspectionClearSiloChain(t *testing.T) {
	silo := &Silo{ID: "silo-02", SafetyState: "clear"}
	err := RecordInspection(silo, "check")
	if !errors.Is(err, ErrSiloRejected) {
		t.Fatalf("clear silo error lost ErrSiloRejected chain: %v", err)
	}
}

func TestRecordInspectionFindingChain(t *testing.T) {
	silo := &Silo{ID: "silo-01", SafetyState: "review"}
	err := RecordInspection(silo, "")
	if !errors.Is(err, ErrFindingEmpty) {
		t.Fatalf("empty finding error lost ErrFindingEmpty chain: %v", err)
	}
}
