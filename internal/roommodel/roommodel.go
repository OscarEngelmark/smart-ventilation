// Package roommodel learns how a room's CO2 responds to the people in it and
// to its damper, from stored readings alone:
//
//	dC/dt = a·N − (b0 + b1·d)·(C − Cout)
//
// It is never given the room's volume or airflows. The model and its
// reasoning are in project_notes.md §7.
package roommodel

import (
	"errors"
	"sort"
	"time"
)

// Sample is one stored value and the time it was taken. A head count or a
// damper level holds until the next one.
type Sample struct {
	Value float64
	Time  time.Time
}

// Model is the room's three learned rates.
type Model struct {
	A  float64 // CO2 rise per person, in ppm per minute
	B0 float64 // share of the gap to outdoor CO2 cleared per minute, damper closed
	B1 float64 // share cleared per minute added by each unit of damper opening
}

// maxGap is the longest step between two CO2 readings still used.
const maxGap = time.Minute

// minIndependence is how far below the diagonal product det(M) may fall
// before the fit is refused.
const minIndependence = 1e-6

// Fit returns the model that best matches the history, by least squares. co2,
// people and damper are the stored CO2 readings (ppm), head counts and damper
// levels (0–1), each in time order; Cout is the outdoor CO2 level, in ppm.
//
// Each step, from one CO2 reading to the next, gives one equation, using the
// head count and damper level in force at the step's start. A step is left out
// when it is longer than maxGap, when the head count or damper level changed
// during it, or when either is unknown that early.
//
// It returns an error when the history can't tell the three rates apart: too
// little of it, nobody ever present, or the damper never moved.
func Fit(co2, people, damper []Sample, Cout float64) (Model, error) {
	var M [3][3]float64 // sums of x·xᵀ over the equations
	var v [3]float64    // sums of x·rate over the equations
	for _, e := range equations(co2, people, damper) {
		excess := e.C - Cout
		x := [3]float64{e.N, -excess, -e.d * excess} // what multiplies a, b0 and b1
		for i := range 3 {
			for j := range 3 {
				M[i][j] += x[i] * x[j]
			}
			v[i] += x[i] * e.rate
		}
	}

	theta, ok := solve(M, v)
	if !ok {
		return Model{}, errors.New("history can't tell the three rates apart")
	}
	return Model{A: theta[0], B0: theta[1], B1: theta[2]}, nil
}

// equation is what one step between CO2 readings says about the room.
type equation struct {
	N    float64 // head count
	d    float64 // damper level
	C    float64 // CO2 over the step, the average of its two readings, in ppm
	rate float64 // change in CO2 over the step, in ppm per minute
}

// equations turns the history into one equation per usable step between CO2
// readings, by the rules in Fit.
func equations(co2, people, damper []Sample) []equation {
	var out []equation
	for i := 0; i+1 < len(co2); i++ {
		first, second := co2[i], co2[i+1]
		dt := second.Time.Sub(first.Time)
		if dt <= 0 || dt > maxGap {
			continue
		}

		N, knowN := valueAt(people, first.Time)
		d, knowD := valueAt(damper, first.Time)
		Nlater, _ := valueAt(people, second.Time)
		dLater, _ := valueAt(damper, second.Time)
		if !knowN || !knowD || Nlater != N || dLater != d {
			continue
		}

		e := equation{
			N:    N,
			d:    d,
			C:    (first.Value + second.Value) / 2,
			rate: (second.Value - first.Value) / dt.Minutes(),
		}
		out = append(out, e)
	}
	return out
}

// valueAt returns the latest value in samples at or before t. ok is false when
// samples has none that early. samples must be in time order.
func valueAt(samples []Sample, t time.Time) (value float64, ok bool) {
	after := sort.Search(len(samples), func(i int) bool {
		return samples[i].Time.After(t)
	}) // index of the first sample later than t
	if after == 0 {
		return 0, false
	}
	return samples[after-1].Value, true
}

// solve returns theta with M·theta = v, by Cramer's rule. For a matrix of sums
// of x·xᵀ, det(M) is at most the product of M's diagonal, and reaches it only
// when the three columns of x vary fully independently; ok is false when it
// falls below minIndependence of that, as the columns are then too close to
// dependent for a trustworthy answer.
func solve(M [3][3]float64, v [3]float64) (theta [3]float64, ok bool) {
	D := det(M)
	if D <= minIndependence*M[0][0]*M[1][1]*M[2][2] {
		return theta, false
	}
	for k := range 3 {
		Mk := M // a copy, since Go copies arrays on assignment
		for i := range 3 {
			Mk[i][k] = v[i]
		}
		theta[k] = det(Mk) / D
	}
	return theta, true
}

func det(M [3][3]float64) float64 {
	return M[0][0]*(M[1][1]*M[2][2]-M[1][2]*M[2][1]) -
		M[0][1]*(M[1][0]*M[2][2]-M[1][2]*M[2][0]) +
		M[0][2]*(M[1][0]*M[2][1]-M[1][1]*M[2][0])
}
