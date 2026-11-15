package srs

import (
	"math"
	"time"
)

type Card struct {
	IntervalDays int
	Ease         float64
	Reps         int
	Lapses       int
}

// Grade: 0 = forgot, 1 = hard, 2 = good, 3 = easy
func Review(c Card, grade int, now time.Time) (Card, time.Time) {
	if grade == 0 {
		c.Lapses++
		c.Reps = 0
		c.IntervalDays = 1
		c.Ease = math.Max(1.3, c.Ease-0.2)
		return c, now.Add(10 * time.Minute) // see it again in this session
	}

	switch c.Reps {
	case 0:
		c.IntervalDays = 1
	case 1:
		c.IntervalDays = 3
	default:
		c.IntervalDays = int(math.Round(float64(c.IntervalDays) * c.Ease))
	}

	switch grade {
	case 1:
		c.Ease = math.Max(1.3, c.Ease-0.15)
		c.IntervalDays = int(math.Max(1, math.Round(float64(c.IntervalDays)*0.8)))
	case 3:
		c.Ease += 0.15
		c.IntervalDays = int(math.Round(float64(c.IntervalDays) * 1.3))
	}

	c.Reps++
	return c, now.AddDate(0, 0, c.IntervalDays)
}
