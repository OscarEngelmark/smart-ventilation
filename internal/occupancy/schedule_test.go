package occupancy

import (
	"testing"
	"time"
)

// capacity of the room the schedule was designed around (A125).
const testCapacity = 6

// at builds a time on Tuesday 2026-09-22, a weekday.
func at(hour, minute int) time.Time {
	return time.Date(2026, 9, 22, hour, minute, 0, 0, time.UTC)
}

// The times checked here are chosen to sit clear of the ±20 minute jitter,
// so the expected count holds whatever offsets the day draws.
func TestPeopleAtKnownTimes(t *testing.T) {
	cases := []struct {
		name string
		when time.Time
		want int
	}{
		{"night", at(3, 0), 0},
		{"before the first arrival", at(6, 0), 0},
		{"mid-morning, regulars in", at(10, 0), 3},
		{"lunch, room empty", at(12, 0), 0},
		{"meeting, room full", at(13, 30), testCapacity},
		{"afternoon, visitors gone", at(15, 0), 3},
		{"evening, everyone left", at(19, 0), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := PeopleAt(c.when, testCapacity); got != c.want {
				t.Errorf("PeopleAt(%s) = %d, want %d", c.when.Format("15:04"), got, c.want)
			}
		})
	}
}

func TestWeekendIsEmpty(t *testing.T) {
	// Saturday and Sunday of the same week, at a time that is busy on a
	// weekday.
	for _, day := range []int{26, 27} {
		when := time.Date(2026, 9, day, 13, 30, 0, 0, time.UTC)
		if got := PeopleAt(when, testCapacity); got != 0 {
			t.Errorf("PeopleAt(%s) = %d, want 0", when.Weekday(), got)
		}
	}
}

// The room must never hold more than it can, and must fill exactly once a
// day, since the meeting is the scenario the CO2 loop is demonstrated on.
func TestDayStaysWithinCapacityAndFillsOnce(t *testing.T) {
	peak := 0
	for minute := 0; minute < 24*60; minute++ {
		got := PeopleAt(at(0, minute), testCapacity)
		if got > testCapacity {
			t.Fatalf("PeopleAt(%02d:%02d) = %d, over capacity %d",
				minute/60, minute%60, got, testCapacity)
		}
		if got > peak {
			peak = got
		}
	}
	if peak != testCapacity {
		t.Errorf("peak occupancy = %d, want the room to fill to %d", peak, testCapacity)
	}
}

// Lunch is meant to empty the room, not thin it out, since that dip is what
// makes the CO2 decay visible before the meeting.
func TestRoomEmptiesAtLunch(t *testing.T) {
	low := testCapacity
	for minute := 11 * 60; minute < 13*60; minute++ {
		if got := PeopleAt(at(0, minute), testCapacity); got < low {
			low = got
		}
	}
	if low > 1 {
		t.Errorf("lowest occupancy over lunch = %d, want at most 1", low)
	}
}

// The offsets must depend on the date alone, so that replaying a day gives
// the same occupancy and a test or demo can be repeated.
func TestJitterDependsOnDateOnly(t *testing.T) {
	morning := dayJitter(at(8, 15), testCapacity)
	evening := dayJitter(at(20, 45), testCapacity)
	for i := range morning {
		if morning[i] != evening[i] {
			t.Fatalf("person %d: jitter differs within the same day: %+v vs %+v",
				i, morning[i], evening[i])
		}
	}

	nextDay := dayJitter(at(8, 15).AddDate(0, 0, 1), testCapacity)
	if morning[0] == nextDay[0] {
		t.Errorf("person 0: jitter is the same on consecutive days: %+v", morning[0])
	}
}

func TestZeroCapacityIsEmpty(t *testing.T) {
	if got := PeopleAt(at(13, 30), 0); got != 0 {
		t.Errorf("PeopleAt with capacity 0 = %d, want 0", got)
	}
}
