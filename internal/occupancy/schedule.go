// Package occupancy turns a moment in time into the number of people in a
// room. The pattern is a weekday office day with a midday meeting; the
// schedule and the reasons behind it are in project_notes.md §6.
//
// The room's capacity is passed in rather than stored here, because it is
// derived from the room's floor area in BuildSim.
package occupancy

import (
	"math/rand/v2"
	"time"
)

// The fixed points of the weekday pattern, as time of day. Regulars come in
// one at a time between arrivalStart and arrivalEnd and leave the same way
// between departStart and departEnd. The lunch window is shared, so the room
// empties around midday rather than thinning out.
const (
	arrivalStart = 7 * time.Hour
	arrivalEnd   = 9 * time.Hour
	lunchStart   = 11*time.Hour + 30*time.Minute
	lunchEnd     = 12*time.Hour + 30*time.Minute
	meetingStart = 13 * time.Hour
	meetingEnd   = 14 * time.Hour
	departStart  = 16 * time.Hour
	departEnd    = 18 * time.Hour
)

// maxJitter is how far one person's own times may sit from the pattern.
const maxJitter = 20 * time.Minute

// personJitter holds one person's offsets for the day.
type personJitter struct {
	arrive   time.Duration
	leave    time.Duration
	lunchOut time.Duration
	lunchIn  time.Duration
}

// PeopleAt returns how many people are in the room at t, where capacity is
// the most the room holds. Half of the capacity, rounded up, are regulars
// who are in for the working day; the rest only join the midday meeting, so
// the room reaches capacity exactly once a day.
func PeopleAt(t time.Time, capacity int) int {
	if capacity <= 0 {
		return 0
	}
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return 0
	}

	regulars := (capacity + 1) / 2
	visitors := capacity - regulars
	jitter := dayJitter(t, capacity)
	since := sinceMidnight(t)

	present := 0
	for i := range regulars {
		if regularPresent(since, i, regulars, jitter[i]) {
			present++
		}
	}
	for i := range visitors {
		j := jitter[regulars+i]
		if since >= meetingStart+j.arrive && since < meetingEnd+j.leave {
			present++
		}
	}
	return present
}

// regularPresent reports whether regular i of n is in the room at time of
// day since.
func regularPresent(since time.Duration, i, n int, j personJitter) bool {
	arrive := arrivalStart + spread(arrivalEnd-arrivalStart, i, n) + j.arrive
	leave := departStart + spread(departEnd-departStart, i, n) + j.leave
	if since < arrive || since >= leave {
		return false
	}
	if since >= lunchStart+j.lunchOut && since < lunchEnd+j.lunchIn {
		return false
	}
	return true
}

// spread places person i of n across a window so that they arrive or leave
// one at a time rather than all at once.
func spread(window time.Duration, i, n int) time.Duration {
	return window * time.Duration(i) / time.Duration(n)
}

// dayJitter draws every person's offsets for the day t falls in. The seed is
// the calendar date, so the same day always plays out the same way and a
// demo or a test can be repeated.
func dayJitter(t time.Time, people int) []personJitter {
	year, month, day := t.Date()
	seed := uint64(year)*10000 + uint64(month)*100 + uint64(day)
	rng := rand.New(rand.NewPCG(seed, 0))

	jitter := make([]personJitter, people)
	for i := range jitter {
		jitter[i] = personJitter{
			arrive:   drawJitter(rng),
			leave:    drawJitter(rng),
			lunchOut: drawJitter(rng),
			lunchIn:  drawJitter(rng),
		}
	}
	return jitter
}

// drawJitter returns an offset between -maxJitter and +maxJitter.
func drawJitter(rng *rand.Rand) time.Duration {
	return time.Duration(rng.Int64N(int64(2*maxJitter+1))) - maxJitter
}

// sinceMidnight is t's time of day, in t's own location.
func sinceMidnight(t time.Time) time.Duration {
	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return t.Sub(midnight)
}
