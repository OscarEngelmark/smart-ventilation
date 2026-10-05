package message

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/schemas"
)

func TestEveryMessageMatchesItsSchema(t *testing.T) {
	ts := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		schema string
		msg    any
	}{
		{"co2_reading.schema.json", CO2Reading{RoomID: "level0/A125", PPM: 650, Ts: ts}},
		{"occupancy_reading.schema.json", OccupancyReading{RoomID: "level0/A125", Count: 6, Ts: ts}},
		{"room_model.schema.json", RoomModel{RoomID: "level0/A125", Date: "2026-10-09", A: 4.7, B0: 0.00875, B1: 0.0496}},
		{"ventilation_command.schema.json", VentilationCommand{RoomID: "level0/A125", Level: 0.4, Ts: ts}},
	}
	for _, c := range cases {
		sch, err := schemas.Load(c.schema)
		if err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(c.msg)
		if err != nil {
			t.Fatal(err)
		}
		if err := schemas.Validate(sch, payload); err != nil {
			t.Errorf("%T doesn't match %s: %v", c.msg, c.schema, err)
		}
	}
}
