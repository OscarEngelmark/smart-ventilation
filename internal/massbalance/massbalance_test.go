package massbalance

import (
	"math"
	"testing"
	"time"
)

const Cout = 420.0 // ppm

func TestSettlesWhereSourceAndClearingBalance(t *testing.T) {
	got := After(Cout, Cout, 28, 0.05, 24*time.Hour) // 28 ppm/min, 5 % per min

	nearly(t, got, Cout+28/0.05, 0.01)
}

func TestWithNothingClearedRisesInAStraightLine(t *testing.T) {
	got := After(500, Cout, 4, 0, 30*time.Minute)

	nearly(t, got, 500+4*30, 0.01)
}

// The solution is exact, so the result must not depend on how t is divided up.
func TestOneLongStepMatchesManyShortOnes(t *testing.T) {
	long := After(Cout, Cout, 28, 0.05, 10*time.Minute)

	short := Cout
	for range 60 {
		short = After(short, Cout, 28, 0.05, 10*time.Second)
	}
	nearly(t, long, short, 0.01)
}

// nearly fails the test unless got is within tol of want.
func nearly(t *testing.T, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("got %.2f ppm, want %.2f ± %.2f", got, want, tol)
	}
}
