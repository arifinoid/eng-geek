package dashboard

import "time"

// Streak returns the number of consecutive study days, counting back from
// today. A streak is still alive when the most recent study day is yesterday,
// so a profile that has not studied yet today keeps yesterday's streak.
// days must be sorted newest first; the time of day is ignored.
func Streak(days []time.Time, now time.Time) int {
	if len(days) == 0 {
		return 0
	}

	today := dateOnly(now)
	latest := dateOnly(days[0])

	expected := today
	switch daysBetween(latest, today) {
	case 0:
		// Studied today: count back from today.
	case 1:
		// Not yet studied today: the streak still holds from yesterday.
		expected = today.AddDate(0, 0, -1)
	default:
		return 0
	}

	streak := 0
	for _, d := range days {
		day := dateOnly(d)
		switch {
		case day.Equal(expected):
			streak++
			expected = expected.AddDate(0, 0, -1)
		case day.After(expected):
			// Duplicate or future entry; nothing to count yet.
		default:
			// A gap ends the streak.
			return streak
		}
	}
	return streak
}

// weekStart returns the Monday of the week containing t.
func weekStart(t time.Time) time.Time {
	t = dateOnly(t)
	weekday := (int(t.Weekday()) + 6) % 7 // Monday is 0.
	return t.AddDate(0, 0, -weekday)
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func sameDay(a, b time.Time) bool {
	return dateOnly(a).Equal(dateOnly(b))
}

func daysBetween(a, b time.Time) int {
	return int(dateOnly(b).Sub(dateOnly(a)).Hours() / 24)
}
