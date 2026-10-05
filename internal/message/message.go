// Package message holds the Go form of each message the services exchange.
// The JSON Schema files in schemas/ define what a valid message is, and each
// type here must match its schema field for field. Reasoning:
// project_notes.md §5.
package message

import "time"

// CO2Reading is the co2_reading message, defined by
// schemas/co2_reading.schema.json.
type CO2Reading struct {
	RoomID string    `json:"room_id"`
	PPM    float64   `json:"ppm"`
	Ts     time.Time `json:"ts"`
}

// OccupancyReading is the occupancy_reading message, defined by
// schemas/occupancy_reading.schema.json.
type OccupancyReading struct {
	RoomID string    `json:"room_id"`
	Count  int       `json:"count"`
	Ts     time.Time `json:"ts"`
}

// RoomModel is the room_model message, defined by
// schemas/room_model.schema.json.
type RoomModel struct {
	RoomID string  `json:"room_id"`
	Date   string  `json:"date"`
	A      float64 `json:"a"`
	B0     float64 `json:"b0"`
	B1     float64 `json:"b1"`
}

// VentilationCommand is the ventilation_command message, defined by
// schemas/ventilation_command.schema.json.
type VentilationCommand struct {
	RoomID string    `json:"room_id"`
	Level  float64   `json:"level"`
	Ts     time.Time `json:"ts"`
}
