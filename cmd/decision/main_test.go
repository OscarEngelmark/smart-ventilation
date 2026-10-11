package main

import (
	"testing"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/message"
	"github.com/OscarEngelmark/smart-ventilation/internal/planner"
)

func TestCO2ReadingWithoutPPMIsDropped(t *testing.T) {
	l := &latest{sch: loadInputSchemas()}
	_, ok := l.keepCO2([]byte(`{"room_id": "level0/A125", "ts": "2026-10-09T12:00:00Z"}`))
	if ok {
		t.Error("a reading without ppm was accepted")
	}
	if l.snapshot().co2 != nil {
		t.Error("a reading without ppm was kept as the latest")
	}
}

func TestValidCO2ReadingIsKept(t *testing.T) {
	l := &latest{sch: loadInputSchemas()}
	r, ok := l.keepCO2([]byte(`{"room_id": "level0/A125", "ppm": 650, "ts": "2026-10-09T12:00:00Z"}`))
	if !ok || r.PPM != 650 {
		t.Fatalf("keepCO2 = %v, %v; want the reading at 650 ppm", r, ok)
	}
	if l.snapshot().co2 == nil {
		t.Error("a valid reading was not kept as the latest")
	}
}

func TestNoReadingOpensDamperFully(t *testing.T) {
	readings := make(chan message.CO2Reading) // no reading ever arrives
	level, _ := nextLevel(readings, 5*time.Minute, 10*time.Millisecond, &latest{}, planner.Settings{}, 0.2)
	if level != 1 {
		t.Errorf("level = %.2f with no reading; want 1, fully open", level)
	}
}

func TestUnchangedReadingLooksFrozenOnlyWithPeople(t *testing.T) {
	l := &latest{sch: loadInputSchemas()}
	l.keepCO2([]byte(`{"room_id": "level0/A125", "ppm": 420, "ts": "2026-10-09T12:00:00Z"}`))
	l.keepCO2([]byte(`{"room_id": "level0/A125", "ppm": 420, "ts": "2026-10-09T12:05:00Z"}`))
	in := l.snapshot()
	in.people = &message.OccupancyReading{Count: 2}
	if !frozen(in, 5*time.Minute) {
		t.Error("420 ppm unchanged for 5 minutes with 2 people did not look frozen")
	}
	in.people = &message.OccupancyReading{Count: 0}
	if frozen(in, 5*time.Minute) {
		t.Error("420 ppm unchanged in an empty room looked frozen")
	}
}
