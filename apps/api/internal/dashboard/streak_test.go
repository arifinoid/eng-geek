package dashboard

import (
	"testing"
	"time"
)

func TestStreak(t *testing.T) {
	now := time.Date(2026, 11, 22, 15, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		days []time.Time
		want int
	}{
		{
			name: "no study days",
			days: nil,
			want: 0,
		},
		{
			name: "studied today",
			days: []time.Time{day(2026, 11, 22)},
			want: 1,
		},
		{
			name: "today and yesterday",
			days: []time.Time{day(2026, 11, 22), day(2026, 11, 21)},
			want: 2,
		},
		{
			name: "three consecutive days",
			days: []time.Time{day(2026, 11, 22), day(2026, 11, 21), day(2026, 11, 20)},
			want: 3,
		},
		{
			name: "not studied today but yesterday keeps the streak",
			days: []time.Time{day(2026, 11, 21)},
			want: 1,
		},
		{
			name: "yesterday and the day before",
			days: []time.Time{day(2026, 11, 21), day(2026, 11, 20)},
			want: 2,
		},
		{
			name: "last study day is two days ago",
			days: []time.Time{day(2026, 11, 20)},
			want: 0,
		},
		{
			name: "a gap breaks the streak",
			days: []time.Time{day(2026, 11, 22), day(2026, 11, 20), day(2026, 11, 19)},
			want: 1,
		},
		{
			name: "duplicate days are ignored",
			days: []time.Time{day(2026, 11, 22), day(2026, 11, 22), day(2026, 11, 21)},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Streak(tt.days, now); got != tt.want {
				t.Fatalf("Streak() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWeekStartIsMonday(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want time.Time
	}{
		{
			name: "sunday",
			in:   time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC),
			want: time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "monday",
			in:   time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC),
			want: time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "wednesday",
			in:   time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC),
			want: time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := weekStart(tt.in); !got.Equal(tt.want) {
				t.Fatalf("weekStart() = %s, want %s", got, tt.want)
			}
		})
	}
}

func day(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 9, 0, 0, 0, time.UTC)
}
