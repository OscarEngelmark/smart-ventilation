// Package planner chooses a room's damper level from its occupancy forecast,
// its head count and its learned room model, by predicting CO2 ahead with
//
//	dC/dt = a·N − (b0 + b1·d)·(C − Cout)
//
// Without a room model, it chooses from the CO2 level alone. The plan and its
// reasoning are in project_notes.md §7.
package planner

import (
	"math"
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
	Target      float64       // CO2 level the plan stays under, in ppm
	LowerMargin float64       // how far under Target a lower level must keep CO2, in ppm
	Horizon     time.Duration // how far ahead the plan looks
	Cout        float64       // outdoor CO2 level, in ppm
	CloseBelow  float64       // CO2 level below which Switch closes the damper, in ppm
}

// steps is how many equal steps the damper range is divided into.
const steps = 20

// dt is the length of one step of the prediction, the CO2 sensor's interval.
const dt = 10 * time.Second

// Level returns the damper level, 0 (closed) to 1 (fully open) in steps of
// 1/steps, for the next reading. Each level is judged by the CO2 it predicts
// over s.Horizon from now, holding that level the whole time. C is the CO2
// level at now, in ppm, and current is the damper's level now.
//
// While current keeps CO2 under s.Target, it is kept, unless a lower level
// keeps CO2 under s.Target − s.LowerMargin; then the lowest such level is
// returned. Once current no longer keeps CO2 under s.Target, the lowest level
// that does is returned, or 1 when no lower level does.
//
// counted is the head count at now, 0 when unknown. When it is above the
// forecast for now rounded up, the extra people are unexpected, and every
// step of the horizon expects at least counted people. A time the forecast
// doesn't cover counts as nobody expected.
func Level(C, current float64, counted int, now time.Time, f Forecast, m roommodel.Model, s Settings) float64 {
	N := expected(f, now, s.Horizon)
	if float64(counted) > math.Ceil(peopleAt(f, now)) {
		for k := range N {
			N[k] = math.Max(N[k], float64(counted))
		}
	}
	if staysUnder(C, current, N, m, s.Cout, s.Target) {
		lower := lowest(C, N, m, s.Cout, s.Target-s.LowerMargin)
		return math.Min(lower, current)
	}
	return lowest(C, N, m, s.Cout, s.Target)
}

// lowest returns the lowest damper level that keeps the predicted CO2 under
// limit, in ppm, or 1 when no lower level does.
func lowest(C float64, N []float64, m roommodel.Model, Cout, limit float64) float64 {
	for i := range steps {
		d := float64(i) / steps
		if staysUnder(C, d, N, m, Cout, limit) {
			return d
		}
	}
	return 1
}

// Switch returns the damper level when there is no room model to plan with,
// from the CO2 level C alone, in ppm: fully open once C reaches s.Target,
// closed once it falls below s.CloseBelow, and current in between.
func Switch(C, current float64, s Settings) float64 {
	if C >= s.Target {
		return 1
	}
	if C < s.CloseBelow {
		return 0
	}
	return current
}

// staysUnder reports whether CO2, starting at C with the damper held at d,
// stays under limit through every step, with N[k] people during step k. C,
// Cout and limit are in ppm.
func staysUnder(C, d float64, N []float64, m roommodel.Model, Cout, limit float64) bool {
	if C >= limit {
		return false
	}
	for _, n := range N {
		rate := m.A*n - (m.B0+m.B1*d)*(C-Cout) // in ppm per minute
		C += rate * dt.Minutes()
		if C >= limit {
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
