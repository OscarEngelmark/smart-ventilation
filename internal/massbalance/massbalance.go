// Package massbalance solves the CO2 balance of one room exactly. Both the
// physical model and the learned room model have this form, with their own
// parameters. Reasoning: project_notes.md §4.
package massbalance

import (
	"math"
	"time"
)

// After returns the CO2 level after t, in ppm, starting at C, for
//
//	dC/dt = S − k·(C − Cout)
//
// held fixed over t. S is the CO2 added per minute, in ppm per minute; k is
// the share of the gap to Cout cleared per minute; C and Cout are in ppm.
func After(C, Cout, S, k float64, t time.Duration) float64 {
	minutes := t.Minutes()
	if k == 0 {
		return C + S*minutes
	}

	// The level C settles at while S and k stay as they are.
	Csteady := Cout + S/k // in ppm

	// The factor the remaining gap to Csteady shrinks by over t.
	gapLeft := math.Exp(-k * minutes)

	return Csteady + (C-Csteady)*gapLeft
}
