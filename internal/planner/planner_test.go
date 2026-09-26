package planner

import (
	"testing"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/roommodel"
)

// a125 is the room model close to what the fit finds for A125.
var a125 = roommodel.Model{A: 4.7, B0: 0.00875, B1: 0.0496}

var settings = Settings{Target: 950, Horizon: time.Hour, Cout: 420}

// at returns hour:minute UTC on Monday 2026-09-28.
func at(hour, minute int) time.Time {
	return time.Date(2026, 9, 28, hour, minute, 0, 0, time.UTC)
}

// day returns a forecast in 5-minute slots over the whole day, with peopleAt
// giving the head count at each time of day.
func day(peopleAt func(t time.Time) float64) Forecast {
	f := Forecast{SlotLength: 5 * time.Minute}
	for t := at(0, 0); t.Before(at(24, 0)); t = t.Add(f.SlotLength) {
		f.Slots = append(f.Slots, Slot{Start: t, N: peopleAt(t)})
	}
	return f
}

// meeting expects six people from 13:00 to 14:00 and nobody otherwise.
func meeting(t time.Time) float64 {
	if t.Hour() == 13 {
		return 6
	}
	return 0
}

// three expects three people all day.
func three(time.Time) float64 {
	return 3
}

func TestEmptyRoomStaysClosed(t *testing.T) {
	empty := day(func(time.Time) float64 { return 0 })

	got := Level(settings.Cout, at(10, 0), empty, a125, settings)

	if got != 0 {
		t.Errorf("level %v, want 0", got)
	}
}

func TestMeetingWithinTheHourOpensBeforeItStarts(t *testing.T) {
	got := Level(settings.Cout, at(12, 30), day(meeting), a125, settings)

	if got == 0 {
		t.Errorf("level 0 half an hour before the meeting, want open")
	}
}

func TestMeetingBeyondTheHourStaysClosed(t *testing.T) {
	got := Level(settings.Cout, at(11, 30), day(meeting), a125, settings)

	if got != 0 {
		t.Errorf("level %v an hour and a half before the meeting, want 0", got)
	}
}

func TestChosenLevelIsTheLowestThatStaysUnder(t *testing.T) {
	now := at(10, 0)
	C := 800.0
	f := day(three)

	d := Level(C, now, f, a125, settings)

	N := expected(f, now, settings.Horizon)
	if !staysUnder(C, d, N, a125, settings) {
		t.Errorf("level %v goes over %v ppm", d, settings.Target)
	}
	lower := d - 1.0/steps
	if lower >= 0 && staysUnder(C, lower, N, a125, settings) {
		t.Errorf("level %v also stays under, want it chosen over %v", lower, d)
	}
}

func TestMeetingNeedsLessWhenTheRoomStartsClean(t *testing.T) {
	clean := Level(settings.Cout, at(13, 0), day(meeting), a125, settings)
	stale := Level(900, at(13, 0), day(meeting), a125, settings)

	if clean >= stale {
		t.Errorf("level %v from outdoor air, %v from 900 ppm, want less from outdoor air", clean, stale)
	}
}

func TestAlreadyOverTheTargetOpensFully(t *testing.T) {
	got := Level(2100, at(10, 0), day(three), a125, settings)

	if got != 1 {
		t.Errorf("level %v at 2100 ppm, want 1", got)
	}
}

func TestTimeOutsideTheForecastExpectsNobody(t *testing.T) {
	f := day(three)

	if got := peopleAt(f, at(24, 30)); got != 0 {
		t.Errorf("%v people the next day, want 0", got)
	}
	if got := peopleAt(f, at(23, 59)); got != 3 {
		t.Errorf("%v people in the last slot, want 3", got)
	}
}
