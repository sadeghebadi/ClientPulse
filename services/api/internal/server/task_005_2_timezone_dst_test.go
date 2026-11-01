package server

import (
	"testing"
	"time"
)

func TestDefaultPreviousMonth_MonthBoundariesAndDST(t *testing.T) {
	testCases := []struct {
		name       string
		tzName     string
		expectedTZ string
	}{
		{
			name:       "Standard UTC",
			tzName:     "UTC",
			expectedTZ: "UTC",
		},
		{
			name:       "US Eastern Time (with DST transitions)",
			tzName:     "America/New_York",
			expectedTZ: "America/New_York",
		},
		{
			name:       "Asia/Tehran timezone",
			tzName:     "Asia/Tehran",
			expectedTZ: "Asia/Tehran",
		},
		{
			name:       "Europe/London (BST / GMT transitions)",
			tzName:     "Europe/London",
			expectedTZ: "Europe/London",
		},
		{
			name:       "Invalid timezone fallback to UTC",
			tzName:     "SolarSystem/Jupiter_Red_Spot",
			expectedTZ: "UTC",
		},
		{
			name:       "Empty timezone fallback to UTC",
			tzName:     "",
			expectedTZ: "UTC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pStart, pEnd, resolvedTZ := DefaultPreviousMonth(tc.tzName)

			if resolvedTZ != tc.expectedTZ {
				t.Errorf("expected resolved timezone %q, got %q", tc.expectedTZ, resolvedTZ)
			}

			loc, err := time.LoadLocation(resolvedTZ)
			if err != nil {
				t.Fatalf("failed loading resolved location: %v", err)
			}

			startInLoc := pStart.In(loc)
			endInLoc := pEnd.In(loc)

			// 1. Period start must be midnight on day 1 of previous month in target timezone
			if startInLoc.Day() != 1 {
				t.Errorf("expected start day 1, got %d", startInLoc.Day())
			}
			if startInLoc.Hour() != 0 || startInLoc.Minute() != 0 || startInLoc.Second() != 0 {
				t.Errorf("expected start at 00:00:00, got %s", startInLoc.Format("15:04:05"))
			}

			// 2. Period end must be strictly after period start
			if !pEnd.After(pStart) {
				t.Errorf("period end %v is not after period start %v", pEnd, pStart)
			}

			// 3. Period end must belong to the exact same month as period start in local time
			if endInLoc.Month() != startInLoc.Month() || endInLoc.Year() != startInLoc.Year() {
				t.Errorf("period end month/year (%v %d) does not match start month/year (%v %d)",
					endInLoc.Month(), endInLoc.Year(), startInLoc.Month(), startInLoc.Year())
			}

			// 4. Adding 1 millisecond to period end must cross into the 1st of the current month
			nextInstant := endInLoc.Add(time.Millisecond)
			if nextInstant.Day() != 1 || nextInstant.Hour() != 0 {
				t.Errorf("expected endOfPrevMonth + 1ms to be midnight of 1st, got day=%d hour=%d",
					nextInstant.Day(), nextInstant.Hour())
			}
		})
	}
}

// TestDSTTransitionExactCalculation verifies that month duration across DST transitions
// preserves local wall-clock calendar boundaries without losing or duplicating hours.
func TestDSTTransitionExactCalculation(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load America/New_York location: %v", err)
	}

	// March 2026 in New York has DST transition (March 8, clocks jump forward 1 hour -> 743 hours in month instead of 744)
	startMarch := time.Date(2026, 3, 1, 0, 0, 0, 0, loc)
	firstOfApril := time.Date(2026, 4, 1, 0, 0, 0, 0, loc)
	endMarch := firstOfApril.Add(-time.Nanosecond)

	if !endMarch.After(startMarch) {
		t.Fatalf("end of March must be after start of March")
	}

	// Verify duration in hours reflects 23-hour day on DST switch
	diffHours := firstOfApril.Sub(startMarch).Hours()
	if diffHours != 743.0 {
		t.Errorf("expected March with DST jump to have 743 hours, got %f", diffHours)
	}

	// November 2026 in New York has DST transition (November 1, clocks fall back 1 hour -> 721 hours)
	startNov := time.Date(2026, 11, 1, 0, 0, 0, 0, loc)
	firstOfDec := time.Date(2026, 12, 1, 0, 0, 0, 0, loc)
	diffNovHours := firstOfDec.Sub(startNov).Hours()
	if diffNovHours != 721.0 {
		t.Errorf("expected November with DST fallback to have 721 hours, got %f", diffNovHours)
	}
}

