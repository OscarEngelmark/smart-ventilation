//go:build e2e

// Package e2e tests the running system from outside, end to end: it pushes the
// room's CO2 above the target in BuildSim and follows the loop's answer.
//
// It needs the whole stack running, so it is left out of a plain `go test ./...`.
// Start the stack at speed 1 with ./start.sh 1, so room time is real time, then run:
//
//	go test -tags e2e -v -count=1 ./test/e2e
//
// -count=1 makes Go run the test every time instead of reusing a cached pass.
//
// The requirements it verifies (FR-1 to FR-3) are in project_notes.md §3.
package e2e

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/buildsim"
	"github.com/OscarEngelmark/smart-ventilation/internal/message"
)

// Addresses and names of the running stack, as docker-compose.yml and sim.env set them.
const (
	buildsimURL = "http://127.0.0.1:9090"
	storageURL  = "http://127.0.0.1:8081"
	roomID      = "level0/A125"
	co2SensorID = "A125-co2"
	damperID    = "A125-damper"
	runEnvPath  = "../../run.env" // written by ./start.sh; go test runs in this package's folder
)

const (
	injectedPPM   = 980 // above the plan's target, below the 1000 ppm threshold
	targetPPM     = 950 // PLAN_TARGET_PPM in sim.env
	fullyOpen     = 1.0
	timeout       = 2 * time.Minute // real time, long enough for a session at speed 1
	responseLimit = 2 * time.Minute // FR-2: room time from the reading to the open damper
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

// TestLoop injects a CO2 level above the target and checks, in order, that the
// reading is stored (FR-1), that the decision service commands the damper
// fully open within responseLimit of the reading and the actuator applies it
// in BuildSim (FR-2), and that CO2 then falls (FR-3). The command is only
// stored once the actuator has accepted it, so a stored command shows the
// decision service received the reading. A failed subtest stops the ones
// after it. The test refuses to run unless the session runs at speed 1.
func TestLoop(t *testing.T) {
	requireSpeedOne(t)
	ctx := context.Background()
	client := buildsim.New(buildsimURL) // the test's link to BuildSim
	since := latestStored(t)

	before, err := client.ActuatorState(ctx, damperID)
	if err != nil {
		t.Fatalf("read the damper from BuildSim: %v", err)
	}
	if before == fullyOpen {
		t.Skip("the damper is already fully open, so no change can be seen; run again later")
	}
	if err := client.SetSensorValue(ctx, co2SensorID, injectedPPM); err != nil {
		t.Fatalf("write %d ppm to BuildSim: %v", injectedPPM, err)
	}
	t.Logf("damper at %.2f; wrote %d ppm to BuildSim", before, injectedPPM)

	var reading message.CO2Reading         // the stored reading of the injected level
	var command message.VentilationCommand // the command the injection triggered
	steps := []struct {
		name  string
		check func(t *testing.T)
	}{
		{"FR-1 reading reaches storage", func(t *testing.T) {
			reading = readingStored(t, since)
		}},
		{"FR-2 decision opens the damper", func(t *testing.T) {
			command = openCommandStored(t, since)
			answeredInTime(t, reading, command)
		}},
		{"FR-2 damper is open in BuildSim", func(t *testing.T) {
			damperOpenInBuildSim(t, ctx, client)
		}},
		{"FR-3 CO2 falls afterwards", func(t *testing.T) {
			co2Falls(t, command.Ts)
		}},
	}
	for _, step := range steps {
		if !t.Run(step.name, step.check) {
			return
		}
	}
}

// requireSpeedOne stops the test unless run.env shows the session running at
// speed 1, where room time and real time run on one clock.
func requireSpeedOne(t *testing.T) {
	env, err := os.ReadFile(runEnvPath)
	if err != nil {
		t.Fatalf("read %s: %v; start the stack with ./start.sh 1", runEnvPath, err)
	}
	for _, line := range strings.Split(string(env), "\n") {
		speed, found := strings.CutPrefix(line, "SIM_SPEED=") // found is false on other lines
		if !found {
			continue
		}
		if speed != "1" {
			t.Fatalf("the session runs at speed %s; start it at speed 1 (./start.sh 1)", speed)
		}
		return
	}
	t.Fatalf("no SIM_SPEED in %s", runEnvPath)
}

// readingStored waits for a stored CO2 reading at or above the target, and
// returns it.
func readingStored(t *testing.T, since time.Time) message.CO2Reading {
	var found message.CO2Reading
	waitFor(t, "stored reading at or above the target", func() bool {
		for _, r := range co2Since(t, since) {
			if r.PPM >= targetPPM {
				found = r
				return true
			}
		}
		return false
	})
	t.Logf("stored %.0f ppm at %s", found.PPM, found.Ts.Format(time.RFC3339))
	return found
}

// answeredInTime checks, from the stored room timestamps, that the command
// came within responseLimit of the reading.
func answeredInTime(t *testing.T, reading message.CO2Reading, command message.VentilationCommand) {
	delay := command.Ts.Sub(reading.Ts)
	t.Logf("command %v of room time after the reading", delay)
	if delay > responseLimit {
		t.Fatalf("command came %v after the reading, want at most %v", delay, responseLimit)
	}
}

// openCommandStored waits for a stored command, newer than since, that opens
// the damper fully, and returns it.
func openCommandStored(t *testing.T, since time.Time) message.VentilationCommand {
	var found message.VentilationCommand
	waitFor(t, "stored command to open the damper fully", func() bool {
		for _, c := range commandsSince(t, since) {
			if c.Ts.After(since) && c.Level == fullyOpen {
				found = c
				return true
			}
		}
		return false
	})
	t.Logf("stored command %.2f at %s", found.Level, found.Ts.Format(time.RFC3339))
	return found
}

// damperOpenInBuildSim checks that BuildSim holds the damper fully open.
func damperOpenInBuildSim(t *testing.T, ctx context.Context, client *buildsim.Client) {
	state, err := client.ActuatorState(ctx, damperID)
	if err != nil {
		t.Fatalf("read the damper from BuildSim: %v", err)
	}
	if state != fullyOpen {
		t.Fatalf("damper at %.2f in BuildSim, want %.2f", state, fullyOpen)
	}
}

// co2Falls waits for three readings after the command and checks that the
// third is lower than the first. The first may still come from a step taken
// before the damper moved; the next two cannot.
func co2Falls(t *testing.T, commandTs time.Time) {
	var after []message.CO2Reading
	waitFor(t, "three readings after the command", func() bool {
		after = nil
		for _, r := range co2Since(t, commandTs) {
			if r.Ts.After(commandTs) {
				after = append(after, r)
			}
		}
		return len(after) >= 3
	})
	first, third := after[0], after[2]
	t.Logf("CO2 %.1f -> %.1f ppm over the readings after the command", first.PPM, third.PPM)
	if third.PPM >= first.PPM {
		t.Fatalf("CO2 rose from %.1f to %.1f ppm with the damper open", first.PPM, third.PPM)
	}
}

// waitFor calls check every half second until it returns true, and fails the
// test if that hasn't happened within timeout. what names the awaited event.
func waitFor(t *testing.T, what string, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("no %s within %v", what, timeout)
}

// latestStored returns the newest room time the storage-service holds.
func latestStored(t *testing.T) time.Time {
	body := get(t, storageURL+"/latest")
	text := strings.TrimSpace(string(body))
	latest, err := time.Parse(time.RFC3339, text)
	if err != nil {
		t.Fatalf("storage-service /latest gave %q; is the stack running? %v", text, err)
	}
	return latest
}

// co2Since returns the room's stored CO2 readings from since on, oldest first.
func co2Since(t *testing.T, since time.Time) []message.CO2Reading {
	var readings []message.CO2Reading
	history(t, "/co2", since, &readings)
	return readings
}

// commandsSince returns the room's stored commands from since on, led by the
// one in force at since, oldest first.
func commandsSince(t *testing.T, since time.Time) []message.VentilationCommand {
	var commands []message.VentilationCommand
	history(t, "/commands", since, &commands)
	return commands
}

// history asks the storage-service for one of the room's stored histories
// from since on, and decodes the JSON answer into out.
func history(t *testing.T, path string, since time.Time, out any) {
	query := url.Values{"room": {roomID}, "since": {since.Format(time.RFC3339)}}
	body := get(t, storageURL+path+"?"+query.Encode())
	if err := json.Unmarshal(body, out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

// get sends a GET request and returns the answer's body, failing the test
// unless the answer is 200 OK.
func get(t *testing.T, address string) []byte {
	t.Helper()
	resp, err := httpClient.Get(address)
	if err != nil {
		t.Fatalf("GET %s: %v; is the stack running?", address, err)
	}
	defer resp.Body.Close() // close the answer when get returns
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("GET %s: read answer: %v", address, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %s", address, resp.Status)
	}
	return body
}
