package occupancyforecast

import (
	"testing"
	"time"
)

const days = 5

// on returns hour:minute UTC on the date offset days after Monday 2026-09-21.
func on(offset, hour, minute int) time.Time {
	return time.Date(2026, 9, 21+offset, hour, minute, 0, 0, time.UTC) // overflowing values roll over
}

// day returns one count per minute over the date offset days after monday,
// with peopleAt giving the count at each hour of the day.
func day(offset int, peopleAt func(hour int) int) []Count {
	var counts []Count
	for m := 0; m < 24*60; m++ {
		t := on(offset, 0, m)
		counts = append(counts, Count{People: peopleAt(t.Hour()), Time: t})
	}
	return counts
}

// meeting has people in the room from 13:00 to 14:00 and nobody otherwise.
func meeting(people int) func(hour int) int {
	return func(hour int) int {
		if hour == 13 {
			return people
		}
		return 0
	}
}

func TestIdenticalDaysGiveThatDaysCount(t *testing.T) {
	var history []Count
	for offset := range 5 { // Monday to Friday
		history = append(history, day(offset, meeting(6))...)
	}
	nextMonday := 7

	expect(t, history, on(nextMonday, 13, 30), 6)
	expect(t, history, on(nextMonday, 10, 0), 0)
}

func TestFewerDaysThanAskedAreAveraged(t *testing.T) {
	history := append(day(0, meeting(2)), day(1, meeting(4))...)

	expect(t, history, on(2, 13, 30), 3)
}

func TestOnlyLatestDaysCount(t *testing.T) {
	history := day(0, meeting(10)) // the oldest weekday, pushed out by the next five
	for offset := 1; offset <= 4; offset++ {
		history = append(history, day(offset, meeting(2))...)
	}
	history = append(history, day(7, meeting(2))...) // the following Monday

	expect(t, history, on(8, 13, 30), 2)
}

func TestWeekendDaysInHistoryAreIgnored(t *testing.T) {
	history := day(4, meeting(2))                    // Friday
	history = append(history, day(5, meeting(6))...) // Saturday

	expect(t, history, on(7, 13, 30), 2)
}

func TestTodayIsLeftOut(t *testing.T) {
	history := append(day(0, meeting(2)), day(1, meeting(6))...)

	expect(t, history, on(1, 13, 30), 2)
}

func TestWeekendIsEmpty(t *testing.T) {
	history := day(4, meeting(6))

	expect(t, history, on(5, 13, 30), 0)
}

func TestTimeOfDayIsReadLocally(t *testing.T) {
	history := day(0, meeting(6)) // stored in UTC
	cest := time.FixedZone("CEST", 2*60*60)

	// 15:30 in UTC+2 is 13:30 UTC, inside the stored meeting.
	expect(t, history, on(1, 13, 30).In(cest), 6)
}

func TestNoHistoryGivesNoForecast(t *testing.T) {
	if _, ok := Expected(nil, on(1, 13, 30), days); ok {
		t.Error("got a forecast from no history, want none")
	}
}

// expect checks the forecast at when against want.
func expect(t *testing.T, history []Count, when time.Time, want float64) {
	t.Helper()
	got, ok := Expected(history, when, days)
	if !ok {
		t.Fatalf("Expected at %s gave no forecast, want %v", when.Format("Mon 15:04"), want)
	}
	if got != want {
		t.Errorf("Expected at %s = %v, want %v", when.Format("Mon 15:04"), got, want)
	}
}
