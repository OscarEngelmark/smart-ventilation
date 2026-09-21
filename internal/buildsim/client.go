// Package buildsim is a REST client for BuildSim, covering only the calls
// this project makes. Its types follow BuildSim's payloads.
package buildsim

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// ErrNoValue means the device exists but holds no value yet, which is how
// BuildSim reports one between its registration and the first write.
var ErrNoValue = errors.New("no value yet")

// Person is one occupant of a room. BuildSim's optional icon and position
// are left out, so its 3D view places the person itself.
type Person struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Alien is part of BuildSim's occupancy payload; always sent empty here.
type Alien struct {
	ID string `json:"id"`
}

// RoomOccupancy is who is in one room.
type RoomOccupancy struct {
	Persons []Person `json:"persons"`
	Aliens  []Alien  `json:"aliens"`
}

// Client talks to one BuildSim instance.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a client for the BuildSim at baseURL, e.g. "http://buildsim:9090".
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// RoomKey names a room the way BuildSim does, e.g. "level0/A125". Room names
// repeat between floors, so the level is always included.
func RoomKey(level, room string) string {
	return level + "/" + room
}

type floorData struct {
	Rooms []struct {
		Name string  `json:"name"`
		Area float64 `json:"area"`
	} `json:"rooms"`
}

// RoomArea returns a room's floor area in m².
func (c *Client) RoomArea(ctx context.Context, level, room string) (float64, error) {
	var floor floorData
	url := fmt.Sprintf("%s/api/building/floors/%s", c.baseURL, level)
	if err := c.do(ctx, http.MethodGet, url, nil, &floor); err != nil {
		return 0, err
	}
	for _, r := range floor.Rooms {
		if r.Name == room {
			return r.Area, nil
		}
	}
	return 0, fmt.Errorf("room %q not found on %s", room, level)
}

// SetOccupancy replaces the occupancy of the whole building: rooms left out
// of occ are emptied, so only one process may call this.
func (c *Client) SetOccupancy(ctx context.Context, occ map[string]RoomOccupancy) error {
	return c.do(ctx, http.MethodPut, c.baseURL+"/api/occupancy", occ, nil)
}

// Occupancy returns every room's occupancy, keyed as RoomKey builds it.
func (c *Client) Occupancy(ctx context.Context) (map[string]RoomOccupancy, error) {
	var occ map[string]RoomOccupancy
	if err := c.do(ctx, http.MethodGet, c.baseURL+"/api/occupancy", nil, &occ); err != nil {
		return nil, err
	}
	return occ, nil
}

type sensorData struct {
	Value string `json:"value"`
}

type actuatorData struct {
	State string `json:"state"`
}

// valueUpdate is BuildSim's payload for writing a sensor value.
type valueUpdate struct {
	DataType string `json:"data_type"`
	Value    string `json:"value"`
}

// SensorValue reads a sensor's value as a number, e.g. CO2 in ppm.
func (c *Client) SensorValue(ctx context.Context, id string) (float64, error) {
	var sensor sensorData
	url := fmt.Sprintf("%s/api/sensors/%s", c.baseURL, id)
	if err := c.do(ctx, http.MethodGet, url, nil, &sensor); err != nil {
		return 0, err
	}
	return asNumber(id, sensor.Value)
}

// SetSensorValue writes a sensor's value, rounded to one decimal.
func (c *Client) SetSensorValue(ctx context.Context, id string, value float64) error {
	url := fmt.Sprintf("%s/api/sensors/%s/value", c.baseURL, id)
	update := valueUpdate{
		DataType: "text", // BuildSim's other option, "binary", is for on/off devices
		Value:    strconv.FormatFloat(value, 'f', 1, 64),
	}
	return c.do(ctx, http.MethodPut, url, update, nil)
}

// ActuatorState reads an actuator's state as a number, e.g. a damper's 0 to 1.
func (c *Client) ActuatorState(ctx context.Context, id string) (float64, error) {
	var actuator actuatorData
	url := fmt.Sprintf("%s/api/actuators/%s", c.baseURL, id)
	if err := c.do(ctx, http.MethodGet, url, nil, &actuator); err != nil {
		return 0, err
	}
	return asNumber(id, actuator.State)
}

// asNumber reads BuildSim's text, keeping "not written yet" apart from text
// that is not a number at all. Callers can recognize the first with
// errors.Is(err, ErrNoValue).
func asNumber(deviceID, text string) (float64, error) {
	if text == "" {
		return 0, fmt.Errorf("%s: %w", deviceID, ErrNoValue)
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a number", deviceID, text)
	}
	return number, nil
}

// do carries out every REST call in this file, so request encoding, status
// checking, and error wording stay in one place. It sends method to url with
// body as a JSON payload when body is not nil, treats any non-2xx status as an
// error, and decodes the JSON reply into out when out is not nil.
func (c *Client) do(ctx context.Context, method, url string, body, out any) error {
	var reader io.Reader // interface http.NewRequest wants for the request body
	if body != nil {
		encoded, err := json.Marshal(body) // Go value -> JSON bytes
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader) // build request
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json") // set JSON header
	}

	resp, err := c.http.Do(req) // send request and get response
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// BuildSim names the offending room key in the body, so keep it.
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: %s: %s", method, url, resp.Status, detail)
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
