package store

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// The time the tests place their readings around.
var noon = time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)

// openTemp is a store in a fresh database file, removed when the test ends.
func openTemp(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := OpenSQLite(filepath.Join(t.TempDir(), "readings.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// save stores one reading, its ppm standing in for which reading it is.
func save(t *testing.T, s *SQLiteStore, room string, ppm float64, ts time.Time) {
	t.Helper()
	r := Reading{RoomID: room, PPM: ppm, Time: ts}
	if err := s.SaveReading(context.Background(), r); err != nil {
		t.Fatalf("save: %v", err)
	}
}

// ppmSince is the ppm of each reading ReadingsSince gives back, in its order.
func ppmSince(t *testing.T, s *SQLiteStore, room string, from time.Time) []float64 {
	t.Helper()
	got, err := s.ReadingsSince(context.Background(), room, from)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var ppm []float64
	for _, r := range got {
		ppm = append(ppm, r.PPM)
	}
	return ppm
}

func TestReadingExactlyAtSinceIsIncluded(t *testing.T) {
	s := openTemp(t)
	save(t, s, "level0/A125", 500, noon)

	want(t, ppmSince(t, s, "level0/A125", noon), []float64{500})
}

func TestReadingBeforeSinceIsLeftOut(t *testing.T) {
	s := openTemp(t)
	save(t, s, "level0/A125", 500, noon.Add(-time.Second))
	save(t, s, "level0/A125", 600, noon)

	want(t, ppmSince(t, s, "level0/A125", noon), []float64{600})
}

// Readings arrive in time order, but nothing guarantees it, and the forecast
// window is only meaningful in order.
func TestReadingsComeBackOldestFirst(t *testing.T) {
	s := openTemp(t)
	save(t, s, "level0/A125", 700, noon.Add(2*time.Minute))
	save(t, s, "level0/A125", 500, noon)
	save(t, s, "level0/A125", 600, noon.Add(time.Minute))

	want(t, ppmSince(t, s, "level0/A125", noon), []float64{500, 600, 700})
}

func TestOtherRoomsAreLeftOut(t *testing.T) {
	s := openTemp(t)
	save(t, s, "level0/A125", 500, noon)
	save(t, s, "level0/B210", 900, noon)

	want(t, ppmSince(t, s, "level0/A125", noon), []float64{500})
}

func TestNoReadingsIsNotAnError(t *testing.T) {
	s := openTemp(t)

	got, err := s.ReadingsSince(context.Background(), "level0/A125", noon)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d readings, want none", len(got))
	}
}

// Timestamps are compared as text, so a since in another zone only matches
// the right rows if it is converted to UTC first.
func TestSinceInAnotherZoneSelectsTheSameReadings(t *testing.T) {
	s := openTemp(t)
	save(t, s, "level0/A125", 500, noon.Add(-time.Hour))
	save(t, s, "level0/A125", 600, noon)

	stockholm := time.FixedZone("CEST", 2*60*60)
	want(t, ppmSince(t, s, "level0/A125", noon.In(stockholm)), []float64{600})
}

func TestReadingComesBackAsItWasSaved(t *testing.T) {
	s := openTemp(t)
	save(t, s, "level0/A125", 543.5, noon)

	got, err := s.ReadingsSince(context.Background(), "level0/A125", noon)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d readings, want 1", len(got))
	}
	if got[0].RoomID != "level0/A125" || got[0].PPM != 543.5 || !got[0].Time.Equal(noon) {
		t.Errorf("got %+v, want level0/A125 543.5 at %v", got[0], noon)
	}
}

// saveCount stores one occupancy count.
func saveCount(t *testing.T, s *SQLiteStore, room string, count int, ts time.Time) {
	t.Helper()
	o := Occupancy{RoomID: room, Count: count, Time: ts}
	if err := s.SaveOccupancy(context.Background(), o); err != nil {
		t.Fatalf("save: %v", err)
	}
}

// The occupancy query mirrors the readings query, so one test covers the
// room, the since bound, and the order together.
func TestOccupancySinceSelectsRoomAndTimeInOrder(t *testing.T) {
	s := openTemp(t)
	saveCount(t, s, "level0/A125", 1, noon.Add(-time.Second))
	saveCount(t, s, "level0/A125", 4, noon.Add(time.Minute))
	saveCount(t, s, "level0/A125", 3, noon)
	saveCount(t, s, "level0/B210", 6, noon)

	got, err := s.OccupancySince(context.Background(), "level0/A125", noon)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var counts []int
	for _, o := range got {
		counts = append(counts, o.Count)
	}
	if !slices.Equal(counts, []int{3, 4}) {
		t.Errorf("got %v, want [3 4]", counts)
	}
}

func TestOccupancyComesBackAsItWasSaved(t *testing.T) {
	s := openTemp(t)
	saveCount(t, s, "level0/A125", 5, noon)

	got, err := s.OccupancySince(context.Background(), "level0/A125", noon)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d counts, want 1", len(got))
	}
	if got[0].RoomID != "level0/A125" || got[0].Count != 5 || !got[0].Time.Equal(noon) {
		t.Errorf("got %+v, want level0/A125 5 at %v", got[0], noon)
	}
}

// want fails the test unless got holds the same ppm values in the same order.
func want(t *testing.T, got, expected []float64) {
	t.Helper()
	if !slices.Equal(got, expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}
