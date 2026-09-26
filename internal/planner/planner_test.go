package planner

import (
	"testing"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/roommodel"
)

// a125 is the room model close to what the fit finds for A125.
var a125 = roommodel.Model{A: 4.7, B0: 0.00875, B1: 0.0496}

var settings = Settings{Target: 950, LowerMargin: 20, Horizon: time.Hour, Cout: 420, CloseBelow: 800}

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

	got := Level(settings.Cout, 0, 0, at(10, 0), empty, a125, settings)

	if got != 0 {
		t.Errorf("level %v, want 0", got)
	}
}

func TestMeetingWithinTheHourOpensBeforeItStarts(t *testing.T) {
	got := Level(settings.Cout, 0, 0, at(12, 30), day(meeting), a125, settings)

	if got == 0 {
		t.Errorf("level 0 half an hour before the meeting, want open")
	}
}

func TestMeetingBeyondTheHourStaysClosed(t *testing.T) {
	got := Level(settings.Cout, 0, 0, at(11, 30), day(meeting), a125, settings)

	if got != 0 {
		t.Errorf("level %v an hour and a half before the meeting, want 0", got)
	}
}

func TestChosenLevelIsTheLowestThatStaysUnder(t *testing.T) {
	now := at(10, 0)
	C := 800.0
	f := day(three)

	d := Level(C, 0, 0, now, f, a125, settings)

	N := expected(f, now, settings.Horizon)
	if !staysUnder(C, d, N, a125, settings.Cout, settings.Target) {
		t.Errorf("level %v goes over %v ppm", d, settings.Target)
	}
	lower := d - 1.0/steps
	if lower >= 0 && staysUnder(C, lower, N, a125, settings.Cout, settings.Target) {
		t.Errorf("level %v also stays under, want it chosen over %v", lower, d)
	}
}

func TestMeetingNeedsLessWhenTheRoomStartsClean(t *testing.T) {
	clean := Level(settings.Cout, 0, 0, at(13, 0), day(meeting), a125, settings)
	stale := Level(900, 0, 0, at(13, 0), day(meeting), a125, settings)

	if clean >= stale {
		t.Errorf("level %v from outdoor air, %v from 900 ppm, want less from outdoor air", clean, stale)
	}
}

func TestAlreadyOverTheTargetOpensFully(t *testing.T) {
	got := Level(2100, 0, 0, at(10, 0), day(three), a125, settings)

	if got != 1 {
		t.Errorf("level %v at 2100 ppm, want 1", got)
	}
}

func TestUnexpectedPeopleRaiseTheLevel(t *testing.T) {
	expected := Level(800, 0, 3, at(10, 0), day(three), a125, settings)
	unexpected := Level(800, 0, 6, at(10, 0), day(three), a125, settings)

	if unexpected <= expected {
		t.Errorf("level %v for 6 people, %v for the 3 forecast, want higher for 6", unexpected, expected)
	}
}

func TestCountAtTheForecastRoundedUpChangesNothing(t *testing.T) {
	f := day(func(time.Time) float64 { return 2.4 })

	counted := Level(800, 0, 3, at(10, 0), f, a125, settings)
	unknown := Level(800, 0, 0, at(10, 0), f, a125, settings)

	if counted != unknown {
		t.Errorf("level %v with 3 counted, %v with no count, want the same", counted, unknown)
	}
}

func TestUnexpectedCountIsHeldUnderTheTarget(t *testing.T) {
	now := at(10, 0)
	C := settings.Cout
	empty := day(func(time.Time) float64 { return 0 })

	d := Level(C, 0, 6, now, empty, a125, settings)

	six := expected(day(func(time.Time) float64 { return 6 }), now, settings.Horizon)
	if !staysUnder(C, d, six, a125, settings.Cout, settings.Target) {
		t.Errorf("level %v lets 6 people take CO2 over %v ppm", d, settings.Target)
	}
	lower := d - 1.0/steps
	if lower >= 0 && staysUnder(C, lower, six, a125, settings.Cout, settings.Target) {
		t.Errorf("level %v also holds 6 people under, want it chosen over %v", lower, d)
	}
}

func TestNoForecastPlansForTheCountedPeople(t *testing.T) {
	none := Level(800, 0, 3, at(10, 0), Forecast{}, a125, settings)
	forecast := Level(800, 0, 3, at(10, 0), day(three), a125, settings)

	if none != forecast {
		t.Errorf("level %v with no forecast, %v with 3 forecast, want the same for 3 counted", none, forecast)
	}
}

// six expects six people all day.
func six(time.Time) float64 {
	return 6
}

func TestLevelStillEnoughIsKeptWhenTheNextOneDownOnlyJustPasses(t *testing.T) {
	now := at(10, 0)
	C := 900.0
	N := expected(day(six), now, settings.Horizon)
	nextDown := 0.90
	if !staysUnder(C, nextDown, N, a125, settings.Cout, settings.Target) ||
		staysUnder(C, nextDown, N, a125, settings.Cout, settings.Target-settings.LowerMargin) {
		t.Fatalf("level %v should keep CO2 under %v ppm but not under %v ppm", nextDown,
			settings.Target, settings.Target-settings.LowerMargin)
	}

	got := Level(C, 0.95, 0, now, day(six), a125, settings)

	if got != 0.95 {
		t.Errorf("level %v, want 0.95 kept", got)
	}
}

func TestFullRoomAtTheTargetSettlesAfterOpeningFully(t *testing.T) {
	now := at(13, 0)
	C := settings.Target
	d := 0.85
	var levels []float64
	for range 360 { // an hour of readings, 10 s apart
		next := Level(C, d, 6, now, day(six), a125, settings)
		if next != d {
			levels = append(levels, next)
		}
		d = next
		rate := a125.A*6 - (a125.B0+a125.B1*d)*(C-settings.Cout) // in ppm per minute
		C += rate * dt.Minutes()
		now = now.Add(dt)
	}

	if len(levels) != 2 || levels[0] != 1 || levels[1] != 0.95 {
		t.Errorf("levels chosen %v, want fully open, then 0.95", levels)
	}
	if C >= settings.Target {
		t.Errorf("CO2 %.1f ppm after an hour, want under %v", C, settings.Target)
	}
}

func TestSwitchOpensAtTheTargetAndClosesBelowTheLowerLevel(t *testing.T) {
	if got := Switch(950, 0, settings); got != 1 {
		t.Errorf("level %v at 950 ppm, want 1", got)
	}
	if got := Switch(799, 1, settings); got != 0 {
		t.Errorf("level %v at 799 ppm, want 0", got)
	}
}

func TestSwitchKeepsTheLevelInBetween(t *testing.T) {
	if got := Switch(900, 0, settings); got != 0 {
		t.Errorf("level %v at 900 ppm while closed, want 0", got)
	}
	if got := Switch(800, 1, settings); got != 1 {
		t.Errorf("level %v at 800 ppm while open, want 1", got)
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
