package domain

import (
	"errors"
	"fmt"
)

var (
	ErrSiloNotFound = errors.New("silo not found")
	ErrSiloRejected = errors.New("silo rejects inspection")
	ErrFindingEmpty = errors.New("finding is required")
)

func RecordInspection(silo *Silo, finding string) error {
	if silo.SafetyState == "clear" {
		return fmt.Errorf("%w: clear silo requires no finding", ErrSiloRejected)
	}
	if finding == "" {
		return fmt.Errorf("%w: finding is required", ErrFindingEmpty)
	}
	silo.LastInspection = finding
	silo.Inspected = true
	return nil
}
