package risk

import (
	"testing"
	"time"
)

func TestHaversineMeters_SamePoint(t *testing.T) {
	if d := HaversineMeters(31.2, 121.5, 31.2, 121.5); d > 1 {
		t.Fatalf("same point distance=%v", d)
	}
}

func TestHaversineMeters_Nearby(t *testing.T) {
	d := HaversineMeters(31.2000, 121.5000, 31.2005, 121.5000)
	if d < 40 || d > 80 {
		t.Fatalf("expected ~55m got %v", d)
	}
	if d >= GPSAlertMeters {
		t.Fatalf("should be under alert threshold, got %v", d)
	}
}

func TestInMinorCurfew(t *testing.T) {
	loc := time.Local
	cases := []struct {
		hour int
		want bool
	}{
		{22, true}, {23, true}, {0, true}, {7, true}, {8, false}, {12, false}, {21, false},
	}
	for _, c := range cases {
		tm := time.Date(2026, 1, 15, c.hour, 0, 0, 0, loc)
		if got := inMinorCurfew(tm); got != c.want {
			t.Fatalf("hour=%d got=%v want=%v", c.hour, got, c.want)
		}
	}
}

func TestAbsMax(t *testing.T) {
	if AbsMax([]float64{-10, 3, 7}) != 10 {
		t.Fatal("AbsMax")
	}
}
