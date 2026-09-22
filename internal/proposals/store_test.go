package proposals

import (
	"testing"
	"time"
)

func TestIsDateExpiredUsesCalendarDateInsteadOfInstant(t *testing.T) {
	validUntil := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	saoPaulo := time.FixedZone("America/Sao_Paulo", -3*60*60)
	now := time.Date(2026, 9, 22, 23, 30, 0, 0, saoPaulo)

	if IsDateExpired(validUntil, now) {
		t.Fatal("proposal valid on the current calendar date must remain available")
	}
}

func TestIsDateExpiredRejectsPreviousCalendarDate(t *testing.T) {
	validUntil := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)

	if !IsDateExpired(validUntil, now) {
		t.Fatal("proposal validity before the current calendar date must be expired")
	}
}
