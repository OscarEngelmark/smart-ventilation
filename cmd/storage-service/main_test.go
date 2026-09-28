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
		{"level0/A125", "co2/A125/reading", true},
		{"level0/B200", "co2/A125/reading", false},
		{"level0/A125", "co2/A1255/reading", false},
		{"level0/A125", "co2/reading", false},
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
	save := storeHandlers(db, sch)["co2/+/reading"]

	save("co2/A125/reading", []byte(`{"room_id": "level0/A125", "ppm": 500, "ts": "2026-10-09T12:00:00Z"}`))
	save("co2/A125/reading", []byte(`{"room_id": "level0/B200", "ppm": 600, "ts": "2026-10-09T12:00:00Z"}`))

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
