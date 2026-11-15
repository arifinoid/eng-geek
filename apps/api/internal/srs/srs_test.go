package srs

import (
	"testing"
	"time"
)

func TestReviewForgotResetsAndRepeatsSoon(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	got, due := Review(Card{IntervalDays: 10, Ease: 2.5, Reps: 3}, 0, now)

	if got.Lapses != 1 {
		t.Fatalf("lapses = %d, want 1", got.Lapses)
	}
	if got.Reps != 0 || got.IntervalDays != 1 {
		t.Fatalf("forgot card = %+v, want reps 0 interval 1", got)
	}
	if !due.Equal(now.Add(10 * time.Minute)) {
		t.Fatalf("due = %v, want now + 10m", due)
	}
	if got.Ease != 2.3 {
		t.Fatalf("ease = %.2f, want 2.30", got.Ease)
	}
}

func TestReviewGoodIntervals(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	c := Card{Ease: 2.5}

	c, due := Review(c, 2, now)
	if c.IntervalDays != 1 || c.Reps != 1 {
		t.Fatalf("first good = %+v, want interval 1 reps 1", c)
	}

	c, due = Review(c, 2, due)
	if c.IntervalDays != 3 || c.Reps != 2 {
		t.Fatalf("second good = %+v, want interval 3 reps 2", c)
	}

	c, _ = Review(c, 2, due)
	if c.IntervalDays != 8 { // round(3 * 2.5) = 8
		t.Fatalf("third good interval = %d, want 8", c.IntervalDays)
	}
}

func TestReviewEasyIncreasesEase(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	got, _ := Review(Card{IntervalDays: 5, Ease: 2.5, Reps: 2}, 3, now)

	if got.Ease != 2.65 {
		t.Fatalf("ease = %.2f, want 2.65", got.Ease)
	}
	// Interval is first scaled by ease, then by the easy factor:
	// round(round(5 * 2.5) * 1.3) = round(13 * 1.3) = 17.
	if got.IntervalDays != 17 {
		t.Fatalf("interval = %d, want 17", got.IntervalDays)
	}
}
