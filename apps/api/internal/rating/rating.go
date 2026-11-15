package rating

import "math"

const (
	StartRating = 800.0
	MinRating   = 400.0
	MaxRating   = 2200.0
)

type Level struct {
	Code string
	Min  float64 // inclusive
	Max  float64 // exclusive
}

var Levels = []Level{
	{"A1", 0, 1000},
	{"A2", 1000, 1200},
	{"B1", 1200, 1400},
	{"B2", 1400, 1600},
	{"C1", 1600, 1800},
	{"C2", 1800, math.Inf(1)},
}

// Expected returns the expected score (0..1) for rating r on difficulty d.
func Expected(r, d float64) float64 {
	return 1 / (1 + math.Pow(10, (d-r)/400))
}

// K returns the K-factor based on how many attempts the skill has had.
func K(attempts int) float64 {
	switch {
	case attempts < 20:
		return 32
	case attempts < 100:
		return 24
	default:
		return 16
	}
}

// Update returns the new rating and the delta. score must be in [0, 1].
func Update(r, d, score float64, attempts int) (newRating, delta float64) {
	score = math.Max(0, math.Min(1, score))
	delta = K(attempts) * (score - Expected(r, d))
	newRating = math.Max(MinRating, math.Min(MaxRating, r+delta))
	return newRating, newRating - r
}

// LevelFor maps a rating to a CEFR level code.
func LevelFor(r float64) string {
	for _, l := range Levels {
		if r >= l.Min && r < l.Max {
			return l.Code
		}
	}
	return "A1"
}
