package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/store"
)

func TestRoomMatchesTopic(t *testing.T) {
	cases := []struct {
		roomID, topic string
		want          bool
	}{
		{"level0/A125", "level0/A125/co2/reading", true},
		{"level0/B200", "level0/A125/co2/reading", false},
		{"level0/A125", "level0/A1255/co2/reading", false},
		{"level1/A125", "level0/A125/co2/reading", false},
	}
	for _, c := range cases {
		if got := roomMatchesTopic(c.roomID, c.topic); got != c.want {
			t.Errorf("roomMatchesTopic(%q, %q) = %v, want %v", c.roomID, c.topic, got, c.want)
		}
	}
}

func TestReadingOnAnotherRoomsTopicIsNotSaved(t *testing.T) {
	db, err := store.OpenSQLite(filepath.Join(t.TempDir(), "readings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sch := payloadSchemas{reading: mustLoadSchema("co2_reading.schema.json")}
	save := storeHandlers(db, sch)["+/+/co2/reading"]

	save("level0/A125/co2/reading", []byte(`{"room_id": "level0/A125", "ppm": 500, "ts": "2026-10-09T12:00:00Z"}`))
	save("level0/A125/co2/reading", []byte(`{"room_id": "level0/B200", "ppm": 600, "ts": "2026-10-09T12:00:00Z"}`))

	for room, want := range map[string]int{"level0/A125": 1, "level0/B200": 0} {
		readings, err := db.CO2ReadingsSince(context.Background(), room, time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		if len(readings) != want {
			t.Errorf("%s: %d readings stored, want %d", room, len(readings), want)
		}
	}
}

func TestReadingFailingItsSchemaIsNotSaved(t *testing.T) {
	db, err := store.OpenSQLite(filepath.Join(t.TempDir(), "readings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sch := payloadSchemas{reading: mustLoadSchema("co2_reading.schema.json")}
	save := storeHandlers(db, sch)["+/+/co2/reading"]

	save("level0/A125/co2/reading", []byte(`{"room_id": "level0/A125", "ppm": 500, "ts": "2026-10-09T12:00:00Z"}`))
	save("level0/A125/co2/reading", []byte(`{"room_id": "level0/A125", "ppm": -5, "ts": "2026-10-09T12:00:10Z"}`))

	readings, err := db.CO2ReadingsSince(context.Background(), "level0/A125", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(readings) != 1 || readings[0].PPM != 500 {
		t.Errorf("stored %+v, want only the 500 ppm reading", readings)
	}
}
