// Package planner chooses a room's damper level from its occupancy forecast
// and its learned room model, by predicting CO2 ahead with
//
//	dC/dt = a·N − (b0 + b1·d)·(C − Cout)
//
// The plan and its reasoning are in project_notes.md §7.
package planner

import (
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/roommodel"
)

// Forecast is the expected head count over a day, in slots of equal length.
type Forecast struct {
	SlotLength time.Duration
	Slots      []Slot
}

// Slot is the expected head count from Start until the slot's end.
type Slot struct {
	Start time.Time
	N     float64
}

// Settings are the plan's fixed values.
type Settings struct {
	Target  float64       // CO2 level the plan stays under, in ppm
	Horizon time.Duration // how far ahead the plan looks
	Cout    float64       // outdoor CO2 level, in ppm
}

// steps is how many equal steps the damper range is divided into.
const steps = 20

// dt is the length of one step of the prediction.
const dt = time.Minute

// Level returns the lowest damper level, 0 (closed) to 1 (fully open) in steps
// of 1/steps, that keeps the predicted CO2 under s.Target for s.Horizon from
// now, holding that level the whole time. When no lower level does, it returns
// 1, whether or not fully open does. C is the CO2 level at now, in ppm. A time
// the forecast doesn't cover counts as nobody expected.
func Level(C float64, now time.Time, f Forecast, m roommodel.Model, s Settings) float64 {
	N := expected(f, now, s.Horizon)
	for i := range steps {
		d := float64(i) / steps
		if staysUnder(C, d, N, m, s) {
			return d
		}
	}
	return 1
}

// staysUnder reports whether CO2, starting at C with the damper held at d,
// stays under s.Target through every step, with N[k] people during step k.
func staysUnder(C, d float64, N []float64, m roommodel.Model, s Settings) bool {
	if C >= s.Target {
		return false
	}
	for _, n := range N {
		rate := m.A*n - (m.B0+m.B1*d)*(C-s.Cout) // in ppm per minute
		C += rate * dt.Minutes()
		if C >= s.Target {
			return false
		}
	}
	return true
}

// expected returns the forecast head count for each step of the horizon,
// taken at the step's start.
func expected(f Forecast, now time.Time, horizon time.Duration) []float64 {
	var N []float64
	for t := now; t.Before(now.Add(horizon)); t = t.Add(dt) {
		N = append(N, peopleAt(f, t))
	}
	return N
}

// peopleAt returns the forecast head count at t, or 0 when no slot covers t.
func peopleAt(f Forecast, t time.Time) float64 {
	for _, slot := range f.Slots {
		end := slot.Start.Add(f.SlotLength)
		if !t.Before(slot.Start) && t.Before(end) {
			return slot.N
		}
	}
	return 0
}
