package co2

import (
	"math"
	"testing"
	"time"
)

// A125's airflow at both ends of the damper, from project_notes.md §6.
const (
	Qshut = 10.43  // L/s, 0.35 per m² of 29.8 m²
	Qopen = 115.86 // L/s, 2 * G*N_max / (1000 - 420)
)

// step advances A125 by dt, so each test only spells out what it is about.
func step(C float64, N int, Q float64, dt time.Duration) float64 {
	const V, G, Cout = 71.5, 0.0056, 420.0 // m³, L/s per person, ppm
	return Step(C, N, V, Q, G, Cout, dt)
}

// settled runs a day, long enough that C has reached its steady level.
func settled(C float64, N int, Q float64) float64 {
	return step(C, N, Q, 24*time.Hour)
}

func TestFullRoomFullyOpenSettlesBelowThreshold(t *testing.T) {
	// 420 + 1e6 * 0.0056*6 / 115.86
	nearly(t, settled(420, 6, Qopen), 710, 1)
}

func TestFullRoomDamperShutSettlesFarAbove(t *testing.T) {
	// 420 + 1e6 * 0.0056*6 / 10.43
	nearly(t, settled(420, 6, Qshut), 3641, 1)
}

func TestOnePersonDamperShutSettlesNearThreshold(t *testing.T) {
	nearly(t, settled(420, 1, Qshut), 957, 1)
}

func TestEmptyRoomReturnsToOutdoor(t *testing.T) {
	nearly(t, settled(2000, 0, Qshut), 420, 1)
}

func TestRoomAtItsSteadyLevelStaysThere(t *testing.T) {
	nearly(t, step(710, 6, Qopen, 10*time.Second), 710, 1)
}

// The step is exact, so the result must not depend on how it is divided up.
func TestOneLongStepMatchesManyShortOnes(t *testing.T) {
	long := step(420, 6, Qopen, 10*time.Minute)

	short := 420.0
	for range 60 {
		short = step(short, 6, Qopen, 10*time.Second)
	}
	nearly(t, long, short, 0.01)
}

// nearly fails the test unless got is within tol of want.
func nearly(t *testing.T, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("got %.1f ppm, want %.1f ± %.1f", got, want, tol)
	}
}
