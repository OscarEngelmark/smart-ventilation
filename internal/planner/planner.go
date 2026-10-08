// Package planner chooses a room's damper level from its head count and its
// learned room model, by predicting CO2 ahead with the model. Without a room
// model, it chooses from the CO2 level alone. The plan and its reasoning are
// in project_notes.md §7.
package planner

import (
	"math"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/roommodel"
)

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

// Level returns the damper level, 0 (closed) to 1 (fully open) in steps of
// 1/steps, for the next reading. Each level is judged by the CO2 it predicts
// over s.Horizon from now, holding that level the whole time. C is the CO2
// level now, in ppm, and current is the damper's level now. counted is the
// head count now, 0 before the first count; the prediction expects that many
// people for the whole horizon.
//
// While current keeps CO2 under s.Target, it is kept, unless a lower level
// keeps CO2 under s.Target − s.LowerMargin; then the lowest such level is
// returned. Once current no longer keeps CO2 under s.Target, the lowest level
// that does is returned, or 1 when no lower level does.
func Level(C, current float64, counted int, m roommodel.Model, s Settings) float64 {
	N := float64(counted)
	if staysUnder(C, current, N, m, s, s.Target) {
		lower := lowest(C, N, m, s, s.Target-s.LowerMargin)
		return math.Min(lower, current)
	}
	return lowest(C, N, m, s, s.Target)
}

// lowest returns the lowest damper level that keeps the predicted CO2 under
// limit, in ppm, or 1 when no lower level does.
func lowest(C, N float64, m roommodel.Model, s Settings, limit float64) float64 {
	for i := range steps {
		d := float64(i) / steps
		if staysUnder(C, d, N, m, s, limit) {
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

// staysUnder reports whether CO2, starting at C with N people and the damper
// held at d, stays under limit through s.Horizon. C and limit are in ppm.
// CO2 moves steadily toward the level it settles at, so it stays under limit
// throughout when it is under limit both now and at the end of s.Horizon.
func staysUnder(C, d, N float64, m roommodel.Model, s Settings, limit float64) bool {
	if C >= limit {
		return false
	}
	return m.Predict(C, N, d, s.Cout, s.Horizon) < limit
}
