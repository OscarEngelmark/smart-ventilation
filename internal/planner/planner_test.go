package planner

import (
	"testing"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/roommodel"
)

// a125 is the room model close to what the fit finds for A125.
var a125 = roommodel.Model{A: 4.7, B0: 0.00875, B1: 0.0496}

var settings = Settings{Target: 950, LowerMargin: 20, Horizon: time.Hour, Cout: 420, CloseBelow: 800}

func TestEmptyRoomStaysClosed(t *testing.T) {
	got := Level(settings.Cout, 0, 0, a125, settings)

	if got != 0 {
		t.Errorf("level %v, want 0", got)
	}
}

func TestChosenLevelIsTheLowestThatHoldsTheCountedPeople(t *testing.T) {
	C := settings.Cout

	d := Level(C, 0, 6, a125, settings)

	if !staysUnder(C, d, 6, a125, settings, settings.Target) {
		t.Errorf("level %v lets 6 people take CO2 over %v ppm", d, settings.Target)
	}
	lower := d - 1.0/steps
	if lower >= 0 && staysUnder(C, lower, 6, a125, settings, settings.Target) {
		t.Errorf("level %v also holds 6 people under, want it chosen over %v", lower, d)
	}
}

func TestAlreadyOverTheTargetOpensFully(t *testing.T) {
	got := Level(2100, 0, 3, a125, settings)

	if got != 1 {
		t.Errorf("level %v at 2100 ppm, want 1", got)
	}
}

func TestMorePeopleRaiseTheLevel(t *testing.T) {
	three := Level(800, 0, 3, a125, settings)
	six := Level(800, 0, 6, a125, settings)

	if six <= three {
		t.Errorf("level %v for 6 people, %v for 3, want higher for 6", six, three)
	}
}

func TestLevelStillEnoughIsKeptWhenTheNextOneDownOnlyJustPasses(t *testing.T) {
	C := 900.0
	nextDown := 0.90
	if !staysUnder(C, nextDown, 6, a125, settings, settings.Target) ||
		staysUnder(C, nextDown, 6, a125, settings, settings.Target-settings.LowerMargin) {
		t.Fatalf("level %v should keep CO2 under %v ppm but not under %v ppm", nextDown,
			settings.Target, settings.Target-settings.LowerMargin)
	}

	got := Level(C, 0.95, 6, a125, settings)

	if got != 0.95 {
		t.Errorf("level %v, want 0.95 kept", got)
	}
}

func TestFullRoomAtTheTargetSettlesAfterOpeningFully(t *testing.T) {
	C := settings.Target
	d := 0.85
	var levels []float64
	for range 360 { // an hour of readings, 10 s apart
		next := Level(C, d, 6, a125, settings)
		if next != d {
			levels = append(levels, next)
		}
		d = next
		C = predict(C, d, 6, a125, settings.Cout, 10*time.Second)
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

// BenchmarkLevelFullRoom times one plan for a full room held near the target,
// the state the room spends a meeting in.
func BenchmarkLevelFullRoom(b *testing.B) {
	for b.Loop() {
		Level(920, 0.9, 6, a125, settings)
	}
}

// BenchmarkPrediction times one prediction over the horizon; a plan makes at
// most steps+1 of them.
func BenchmarkPrediction(b *testing.B) {
	for b.Loop() {
		staysUnder(settings.Cout, 1, 6, a125, settings, settings.Target)
	}
}
