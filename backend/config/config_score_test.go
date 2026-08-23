package config

import "testing"

func TestConfigMaxHistoryHonorsEnv(t *testing.T) {
	t.Setenv("MAX_HISTORY", "100")
	if got := MaxHistory(); got != 100 {
		t.Fatalf("MaxHistory()=%d want 100 from MAX_HISTORY env", got)
	}
}

func TestConfigMaxWorkersHonorsEnv(t *testing.T) {
	t.Setenv("MAX_INSPECTION_WORKERS", "5")
	if got := MaxInspectionWorkers(); got != 5 {
		t.Fatalf("MaxInspectionWorkers()=%d want 5 from MAX_INSPECTION_WORKERS env", got)
	}
}
