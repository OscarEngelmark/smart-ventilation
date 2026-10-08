// Package co2 is the physical model of the CO2 level in one room: how it
// rises as people breathe and falls as ventilation air flows through. The
// model and its parameters are in project_notes.md §6.
package co2

import (
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/massbalance"
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
	Vliters := V * 1000                      // in liters
	S := 1e6 * G * float64(N) * 60 / Vliters // CO2 added, in ppm per minute
	k := Q * 60 / Vliters                    // share of the air replaced per minute
	return massbalance.After(C, Cout, S, k, dt)
}
