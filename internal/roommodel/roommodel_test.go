package roommodel

import (
	"math"
	"testing"
	"time"

	"github.com/OscarEngelmark/smart-ventilation/internal/co2"
)

// A125's values in the physical model (project_notes.md §6); only the test knows them.
const (
	V    = 71.5   // m³
	G    = 0.0056 // L/s per person
	Cout = 420.0  // ppm
	Qmin = 10.43  // L/s, damper closed
	Qmax = 69.5   // L/s, damper fully open
)

var start = time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)

// at returns the moment d after the start of the test history.
func at(d time.Duration) time.Time {
	return start.Add(d)
}

// simulate runs the physical model's mass balance for the given minutes, with
// peopleAt and damperAt giving the head count and damper level in each minute.
// It returns what storage would hold: a CO2 reading every 10 s, a head count
// every minute, and a damper level whenever it changes.
func simulate(minutes int, peopleAt func(minute int) int,
	damperAt func(minute int) float64) (readings, people, damper []Sample) {
	const dt = 10 * time.Second
	C := Cout
	for k := range minutes * 6 {
		t := at(time.Duration(k) * dt)
		m := k / 6
		N, d := peopleAt(m), damperAt(m)

		readings = append(readings, Sample{Value: C, Time: t})
		if k%6 == 0 {
			people = append(people, Sample{Value: float64(N), Time: t})
		}
		if k == 0 || (k%6 == 0 && d != damperAt(m-1)) {
			damper = append(damper, Sample{Value: d, Time: t})
		}

		Q := Qmin + d*(Qmax-Qmin)
		C = co2.Step(C, N, V, Q, G, Cout, dt)
	}
	return readings, people, damper
}

// changingPeople steps from 0 to 6 people and back to 0, half an hour each.
func changingPeople(minute int) int {
	return (minute / 30) % 7
}

// cyclingDamper moves the damper between four levels, 20 minutes each.
func cyclingDamper(minute int) float64 {
	levels := []float64{0, 0.5, 1, 0.25}
	return levels[(minute/20)%len(levels)]
}

func TestFitRecoversTheRoom(t *testing.T) {
	readings, people, damper := simulate(8*60, changingPeople, cyclingDamper)

	got, err := Fit(readings, people, damper, Cout)
	if err != nil {
		t.Fatal(err)
	}

	liters := V * 1000
	near(t, "a", got.A, 1e6*G/liters*60)         // in ppm per minute per person
	near(t, "b0", got.B0, Qmin/liters*60)        // per minute
	near(t, "b1", got.B1, (Qmax-Qmin)/liters*60) // per minute
}

func TestDamperThatNeverMovedGivesNoModel(t *testing.T) {
	halfOpen := func(int) float64 { return 0.5 }
	readings, people, damper := simulate(8*60, changingPeople, halfOpen)

	if _, err := Fit(readings, people, damper, Cout); err == nil {
		t.Error("got a model from a damper that never moved, want an error")
	}
}

func TestEmptyRoomGivesNoModel(t *testing.T) {
	nobody := func(int) int { return 0 }
	readings, people, damper := simulate(8*60, nobody, cyclingDamper)

	if _, err := Fit(readings, people, damper, Cout); err == nil {
		t.Error("got a model from a room nobody entered, want an error")
	}
}

func TestPairAcrossAGapIsLeftOut(t *testing.T) {
	readings := []Sample{{420, at(0)}, {430, at(10 * time.Second)}, {600, at(5 * time.Minute)}}
	people := []Sample{{3, at(0)}}
	damper := []Sample{{0, at(0)}}

	expectEquations(t, readings, people, damper, 1)
}

func TestPairAcrossAHeadCountChangeIsLeftOut(t *testing.T) {
	readings := []Sample{{420, at(0)}, {430, at(10 * time.Second)}, {440, at(20 * time.Second)}}
	people := []Sample{{3, at(0)}, {4, at(10 * time.Second)}}
	damper := []Sample{{0, at(0)}}

	expectEquations(t, readings, people, damper, 1)
}

func TestPairAcrossADamperChangeIsLeftOut(t *testing.T) {
	readings := []Sample{{420, at(0)}, {430, at(10 * time.Second)}, {440, at(20 * time.Second)}}
	people := []Sample{{3, at(0)}}
	damper := []Sample{{0, at(0)}, {1, at(10 * time.Second)}}

	expectEquations(t, readings, people, damper, 1)
}

func TestPairBeforeTheFirstHeadCountIsLeftOut(t *testing.T) {
	readings := []Sample{{420, at(0)}, {430, at(10 * time.Second)}, {440, at(20 * time.Second)}}
	people := []Sample{{3, at(10 * time.Second)}}
	damper := []Sample{{0, at(0)}}

	expectEquations(t, readings, people, damper, 1)
}

// expectEquations checks how many equations the history gives.
func expectEquations(t *testing.T, readings, people, damper []Sample, want int) {
	t.Helper()
	if got := len(equations(readings, people, damper)); got != want {
		t.Errorf("got %d equations, want %d", got, want)
	}
}

// near fails the test unless got is within 0.1% of want.
func near(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.001*math.Abs(want) {
		t.Errorf("%s = %.5g, want %.5g", name, got, want)
	}
}
