package main

import "testing"

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
