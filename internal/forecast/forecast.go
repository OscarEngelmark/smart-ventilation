// Package forecast predicts a room's CO2 level a fixed time ahead, assuming
// nothing in the room changes in the meantime. The model and its reasoning
// are in project_notes.md §7.
package forecast

import "time"

// Reading is one CO2 measurement.
type Reading struct {
	PPM  float64
	Time time.Time
}

// minSpan is the least stretch of time the window must cover to forecast.
const minSpan = 5 * time.Minute

// Predict fits a straight line to the readings from the last window before
// the newest reading, and returns the level that line reaches horizon after
// the newest reading, in ppm. Time is measured in seconds relative to the
// newest reading, so the newest reading is at 0 and the forecast at horizon.
//
// ok is false when the readings in the window span less than minSpan, which
// is too little to tell a trend from noise. The readings may come in any
// order.
func Predict(readings []Reading, window, horizon time.Duration) (ppm float64, ok bool) {
	if len(readings) == 0 {
		return 0, false
	}

	newest := readings[0].Time
	for _, r := range readings {
		if r.Time.After(newest) {
			newest = r.Time
		}
	}
	start := newest.Add(-window)

	var x, y []float64
	oldest := newest
	for _, r := range readings {
		if r.Time.Before(start) {
			continue
		}
		x = append(x, r.Time.Sub(newest).Seconds()) // zero or negative
		y = append(y, r.PPM)
		if r.Time.Before(oldest) {
			oldest = r.Time
		}
	}
	if newest.Sub(oldest) < minSpan {
		return 0, false
	}

	a, b := fitLine(x, y)
	return a + b*horizon.Seconds(), true
}

// fitLine returns the intercept a and slope b of the least-squares line
// y = a + b*x through the points. x must hold at least two distinct values.
func fitLine(x, y []float64) (a, b float64) {
	xMean, yMean := mean(x), mean(y)
	var sxy, sxx float64
	for i := range x {
		sxy += (x[i] - xMean) * (y[i] - yMean)
		sxx += (x[i] - xMean) * (x[i] - xMean)
	}
	b = sxy / sxx
	a = yMean - b*xMean
	return a, b
}

// mean returns the average of v, which must not be empty.
func mean(v []float64) float64 {
	var sum float64
	for _, x := range v {
		sum += x
	}
	return sum / float64(len(v))
}
