// Package room holds a room's physical parameters, each derived from the
// floor area BuildSim gives for it. See project_notes.md §6.
package room

import "math"

// Capacity is how many people a room of this floor area holds.
func Capacity(floorArea, areaPerPerson float64) int {
	return int(math.Round(floorArea / areaPerPerson))
}

// Volume is the room's air volume in m³, from its floor area in m².
func Volume(floorArea, ceilingHeight float64) float64 {
	return floorArea * ceilingHeight
}

// MinAirflow (Q_min) is the airflow in L/s with the damper closed.
func MinAirflow(floorArea, perArea float64) float64 {
	return floorArea * perArea
}

// MaxAirflow (Q_max) is the airflow in L/s with the damper fully open.
func MaxAirflow(floorArea, areaPerPerson, G, Cthres, Cout, margin float64) float64 {
	Nmax := Capacity(floorArea, areaPerPerson)
	justEnough := 1e6 * G * float64(Nmax) / (Cthres - Cout) // holds Nmax at Cthres
	return margin * justEnough
}

// Airflow (Q) is the airflow in L/s at this damper position, where 0 is
// closed and 1 fully open. Positions outside that range are clamped.
func Airflow(damper, Qmin, Qmax float64) float64 {
	open := math.Min(math.Max(damper, 0), 1)
	return Qmin + open*(Qmax-Qmin)
}
