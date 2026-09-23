package forecast

import (
	"math"
	"testing"
	"time"
)

const (
	window  = 10 * time.Minute
	horizon = 30 * time.Minute
)

var t0 = time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)

// series returns one reading every 10 s over span, ending at t0, with
// ppmAt giving the level at each time in minutes before t0 (zero or negative).
func series(span time.Duration, ppmAt func(min float64) float64) []Reading {
	var readings []Reading
	for d := -span; d <= 0; d += 10 * time.Second {
		readings = append(readings, Reading{PPM: ppmAt(d.Minutes()), Time: t0.Add(d)})
	}
	return readings
}

func TestRisingLineContinues(t *testing.T) {
	// 800 ppm now, rising 5 ppm per minute: 800 + 5*30 in half an hour.
	readings := series(window, func(min float64) float64 { return 800 + 5*min })
	got, ok := Predict(readings, window, horizon)
	if !ok {
		t.Fatal("got no forecast, want one")
	}
	nearly(t, got, 950, 0.01)
}

func TestFlatSeriesStaysLevel(t *testing.T) {
	readings := series(window, func(float64) float64 { return 600 })
	got, ok := Predict(readings, window, horizon)
	if !ok {
		t.Fatal("got no forecast, want one")
	}
	nearly(t, got, 600, 0.01)
}

func TestOnlyReadingsInsideWindowCount(t *testing.T) {
	// A steep fall before the window, then a flat 600 inside it.
	readings := series(2*window, func(min float64) float64 {
		if min < -window.Minutes() {
			return 600 - 50*(min+window.Minutes())
		}
		return 600
	})
	got, ok := Predict(readings, window, horizon)
	if !ok {
		t.Fatal("got no forecast, want one")
	}
	nearly(t, got, 600, 0.01)
}

func TestTooLittleDataGivesNoForecast(t *testing.T) {
	readings := series(minSpan-time.Second, func(float64) float64 { return 600 })
	if _, ok := Predict(readings, window, horizon); ok {
		t.Error("got a forecast, want none")
	}
}

func TestNoReadingsGivesNoForecast(t *testing.T) {
	if _, ok := Predict(nil, window, horizon); ok {
		t.Error("got a forecast, want none")
	}
}

// nearly fails the test unless got is within tol of want.
func nearly(t *testing.T, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("got %.2f ppm, want %.2f ± %.2f", got, want, tol)
	}
}
