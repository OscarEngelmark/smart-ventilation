// Package co2 is the physical model of the CO2 level in one room: how it
// rises as people breathe and falls as ventilation air flows through. The
// model and its parameters are in project_notes.md §6.
package co2

import (
	"math"
	"time"
)

// Step advances the mass balance V*dC/dt = G*N - Q*(C - Cout) by dt and
// returns the room's CO2 level at the end of the step, in ppm.
//
//	C    CO2 level in the room now, in ppm
//	N    people in the room
//	V    the room's air volume, in m³
//	Q    airflow through the room, in liters per second
//	G    CO2 one person breathes out, in liters per second
//	Cout CO2 level of the incoming outdoor air, in ppm
//
// N and Q are held fixed over the step, so dt should be set accordingly
func Step(C float64, N int, V, Q, G, Cout float64, dt time.Duration) float64 {
	// The level the room settles at while N and Q stay as they are.
	Csteady := Cout + 1e6*G*float64(N)/Q // in ppm

	// The factor the remaining gap to Csteady shrinks by over dt.
	gapLeft := math.Exp(-Q * dt.Seconds() / (V * 1000)) // V in liters

	return Csteady + (C-Csteady)*gapLeft
}
