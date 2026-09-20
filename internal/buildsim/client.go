// Package buildsim is a small REST client for BuildSim, the shared
// building-state backend. It covers only the calls this project makes.
// BuildSim is a fixed external dependency and is never modified, so this
// package follows its payloads rather than defining its own.
package buildsim

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Person is one occupant of a room. BuildSim also accepts an icon and a
// position; both are optional and left out here, so the 3D view places the
// person itself.
type Person struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Alien is part of BuildSim's occupancy payload. Nothing in this project
// produces one, but the field is always sent as an empty list.
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

// New returns a client for the BuildSim at baseURL, e.g.
// "http://buildsim:9090".
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// RoomKey is how BuildSim names a room wherever a whole collection is
// written, e.g. "level0/A125". Room names repeat between floors, so the
// level is always included.
func RoomKey(level, room string) string {
	return level + "/" + room
}

type floorData struct {
	Rooms []struct {
		Name string  `json:"name"`
		Area float64 `json:"area"`
	} `json:"rooms"`
}

// RoomArea returns a room's floor area in m². Room volume and the number of
// people a room holds are both derived from it, so it is read from BuildSim
// rather than configured per room.
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

// SetOccupancy replaces the occupancy of the whole building. Every room left
// out of occ is emptied, so only one process may call this.
func (c *Client) SetOccupancy(ctx context.Context, occ map[string]RoomOccupancy) error {
	url := c.baseURL + "/api/occupancy"
	return c.do(ctx, http.MethodPut, url, occ, nil)
}

// Occupancy returns the occupancy of every room, keyed as RoomKey builds it.
func (c *Client) Occupancy(ctx context.Context) (map[string]RoomOccupancy, error) {
	var occ map[string]RoomOccupancy
	url := c.baseURL + "/api/occupancy"
	if err := c.do(ctx, http.MethodGet, url, nil, &occ); err != nil {
		return nil, err
	}
	return occ, nil
}

// do sends one request, encoding body as JSON when it is not nil and
// decoding the response into out when out is not nil.
func (c *Client) do(ctx context.Context, method, url string, body, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// BuildSim explains a rejected request in the body, and its
		// occupancy errors name the offending room key, so it is worth
		// keeping in the message.
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
