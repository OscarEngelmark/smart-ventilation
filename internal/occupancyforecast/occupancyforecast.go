// Package occupancyforecast predicts how many people a room holds at a given
// moment, from the head counts stored on recent weekdays. It learns only from
// those counts, never from the schedule that produced them. The model and its
// reasoning are in project_notes.md §7.
package occupancyforecast

import (
	"sort"
	"time"
)

// Count is one stored head count.
type Count struct {
	People int
	Time   time.Time
}

// slot is how finely the day is divided; counts in one slot are averaged.
const slot = 5 * time.Minute

// Expected returns the average head count in the slot of the day that at falls
// in, over the latest days weekdays in history before at's own date. With
// fewer weekdays stored, it averages over those there are. Times of day are
// read in at's time zone, so history stored in UTC lines up with local hours.
//
// At weekends the room is empty, so the result is 0. ok is false when history
// holds no weekday count in that slot to average.
func Expected(history []Count, at time.Time, days int) (people float64, ok bool) {
	if isWeekend(at) {
		return 0, true
	}
	loc := at.Location()
	recent := latestWeekdays(history, dateOf(at), days, loc)
	atSlot := slotOf(at)

	var sum float64
	n := 0
	for _, c := range history {
		t := c.Time.In(loc)
		if !recent[dateOf(t)] || slotOf(t) != atSlot {
			continue
		}
		sum += float64(c.People)
		n++
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

// latestWeekdays returns the dates of the latest days weekdays in history
// before the date before, as a set.
func latestWeekdays(history []Count, before string, days int, loc *time.Location) map[string]bool {
	seen := map[string]bool{}
	var dates []string
	for _, c := range history {
		t := c.Time.In(loc)
		date := dateOf(t)
		if isWeekend(t) || date >= before || seen[date] {
			continue
		}
		seen[date] = true
		dates = append(dates, date)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(dates))) // newest first
	recent := map[string]bool{}
	for i, date := range dates {
		if i == days {
			break
		}
		recent[date] = true
	}
	return recent
}

// dateOf is t's calendar date as "2006-01-02" text, which sorts in time order.
func dateOf(t time.Time) string {
	return t.Format(time.DateOnly)
}

// slotOf numbers the slot of the day t falls in, counting from midnight.
func slotOf(t time.Time) int {
	clock := time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute
	return int(clock / slot)
}

func isWeekend(t time.Time) bool {
	return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}
