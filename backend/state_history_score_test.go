package main

import (
	"example.com/grain-silo-safety-service/config"
	"testing"
)

func TestStateHistoryBoundedByConfig(t *testing.T) {
	m := newOpsStateMachine()
	limit := config.MaxHistory()
	for i := 0; i < limit*4; i++ {
		var err error
		if i%2 == 0 {
			err = m.Move(OpsStatusActive, OpsStatusPaused, "operator")
		} else {
			err = m.Move(OpsStatusPaused, OpsStatusActive, "operator")
		}
		if err != nil {
			t.Fatalf("move %d failed: %v", i, err)
		}
	}
	history := m.History()
	if len(history) != limit {
		t.Fatalf("state history length=%d want capped at %d (unbounded growth)", len(history), limit)
	}
}

func TestStateHistorySnapshotIndependent(t *testing.T) {
	m := newOpsStateMachine()
	if err := m.Move(OpsStatusActive, OpsStatusPaused, "operator"); err != nil {
		t.Fatal(err)
	}
	history := m.History()
	history[0].Reason = "tampered"
	again := m.History()
	if again[0].Reason != "operator" {
		t.Fatalf("state history snapshot was mutated by caller: %v", again[0])
	}
}
