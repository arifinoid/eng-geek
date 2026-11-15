package rating

import (
	"math"
	"testing"
)

func TestUpdateWorkedExample(t *testing.T) {
	newR, delta := Update(800, 900, 0.8, 5)
	if math.Abs(delta-14.1) > 0.2 {
		t.Fatalf("delta = %.2f, want ~14.1", delta)
	}
	if math.Abs(newR-814.1) > 0.2 {
		t.Fatalf("newR = %.2f, want ~814.1", newR)
	}
}

func TestLevelFor(t *testing.T) {
	cases := map[float64]string{
		800: "A1", 999.9: "A1", 1000: "A2", 1399: "B1", 1400: "B2", 1800: "C2",
	}
	for r, want := range cases {
		if got := LevelFor(r); got != want {
			t.Errorf("LevelFor(%.1f) = %s, want %s", r, got, want)
		}
	}
}

func TestUpdateScoreIsClamped(t *testing.T) {
	r, _ := Update(800, 900, 1.5, 5)
	if r > MaxRating {
		t.Fatalf("rating = %.2f, want <= %.2f", r, MaxRating)
	}
}

func TestUpdateFloor(t *testing.T) {
	r, _ := Update(MinRating, 2200, 0, 500)
	if r < MinRating {
		t.Fatalf("rating = %.2f, want >= %.2f", r, MinRating)
	}
}
