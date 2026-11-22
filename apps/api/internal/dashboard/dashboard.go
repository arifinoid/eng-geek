// Package dashboard assembles the per-profile overview: streak, study time,
// skill ratings, and the next activities worth attempting.
package dashboard

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinoid/eng-geek/apps/api/internal/rating"
	"github.com/arifinoid/eng-geek/apps/api/internal/store"
)

// DefaultWeeks is how many weekly study-minute buckets the dashboard returns.
const DefaultWeeks int32 = 8

// MaxWeeks caps the requested history so a caller cannot ask for everything.
const MaxWeeks int32 = 52

// ErrProfileNotFound is returned when the profile id does not exist.
var ErrProfileNotFound = errors.New("profile not found")

// SkillOrder is the display order for the four skills.
var SkillOrder = []store.Skill{
	store.SkillListening,
	store.SkillReading,
	store.SkillWriting,
	store.SkillSpeaking,
}

// nextPerSkill is how many activities are suggested per skill.
const nextPerSkill int32 = 3

// Service reads the queries the dashboard is composed from.
type Service struct {
	queries *store.Queries
}

// NewService builds a dashboard service.
func NewService(queries *store.Queries) *Service {
	return &Service{queries: queries}
}

// SkillRating is one row of the per-skill summary.
type SkillRating struct {
	Skill         string
	Rating        float64
	Level         string
	AttemptsCount int32
}

// WeeklyMinutes is study time for one ISO week.
type WeeklyMinutes struct {
	WeekStart time.Time
	Minutes   int32
}

// NextActivity is a suggested activity with enough context to render it.
type NextActivity struct {
	ID           int64
	UnitID       int64
	UnitPosition int32
	UnitTitle    string
	Type         string
	Skill        string
	Difficulty   int32
	Position     int32
	rating       float64
}

// Response is the whole dashboard payload.
type Response struct {
	Profile        store.Profile
	Level          string
	StreakDays     int
	WeekMinutes    int32
	Skills         []SkillRating
	WeeklyMinutes  []WeeklyMinutes
	NextActivities []NextActivity
}

// Get builds the dashboard for a profile. weeks <= 0 falls back to DefaultWeeks.
func (s *Service) Get(ctx context.Context, profileID int64, weeks int32) (Response, error) {
	if weeks <= 0 {
		weeks = DefaultWeeks
	}
	if weeks > MaxWeeks {
		weeks = MaxWeeks
	}

	profile, err := s.queries.GetProfile(ctx, profileID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Response{}, ErrProfileNotFound
	}
	if err != nil {
		return Response{}, fmt.Errorf("get profile: %w", err)
	}

	tz := profile.Timezone
	if tz == "" {
		tz = "UTC"
	}
	now := time.Now().In(location(tz))

	level, err := s.currentLevel(ctx, profileID)
	if err != nil {
		return Response{}, err
	}

	skills, err := s.skills(ctx, profileID)
	if err != nil {
		return Response{}, err
	}

	days, err := s.queries.StudyDays(ctx, store.StudyDaysParams{Tz: tz, ProfileID: profileID})
	if err != nil {
		return Response{}, fmt.Errorf("study days: %w", err)
	}
	studyDays := make([]time.Time, 0, len(days))
	for _, d := range days {
		studyDays = append(studyDays, d.Time)
	}

	weekly, err := s.queries.WeeklyStudyMinutes(ctx, store.WeeklyStudyMinutesParams{
		Tz:        tz,
		ProfileID: profileID,
		Weeks:     weeks,
	})
	if err != nil {
		return Response{}, fmt.Errorf("weekly study minutes: %w", err)
	}
	weeklyMinutes := make([]WeeklyMinutes, 0, len(weekly))
	for _, w := range weekly {
		weeklyMinutes = append(weeklyMinutes, WeeklyMinutes{WeekStart: w.WeekStart.Time, Minutes: w.Minutes})
	}

	next, err := s.nextActivities(ctx, profileID, profile.Track, level, skills)
	if err != nil {
		return Response{}, err
	}

	return Response{
		Profile:        profile,
		Level:          level,
		StreakDays:     Streak(studyDays, now),
		WeekMinutes:    minutesForWeek(weeklyMinutes, weekStart(now)),
		Skills:         skills,
		WeeklyMinutes:  weeklyMinutes,
		NextActivities: next,
	}, nil
}

// currentLevel returns the highest level the profile has in progress, defaulting
// to A1 for a profile that has not started anything yet.
func (s *Service) currentLevel(ctx context.Context, profileID int64) (string, error) {
	progress, err := s.queries.GetLevelProgress(ctx, profileID)
	if err != nil {
		return "", fmt.Errorf("level progress: %w", err)
	}
	level := rating.Levels[0].Code
	for _, p := range progress {
		if p.Status == store.ProgressStatusInProgress {
			level = p.LevelCode
		}
	}
	return level, nil
}

// skills returns all four skills, filling in the start rating for skills the
// profile has not attempted yet.
func (s *Service) skills(ctx context.Context, profileID int64) ([]SkillRating, error) {
	rows, err := s.queries.GetSkillRatings(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("skill ratings: %w", err)
	}
	bySkill := make(map[store.Skill]store.GetSkillRatingsRow, len(rows))
	for _, r := range rows {
		bySkill[r.Skill] = r
	}

	result := make([]SkillRating, 0, len(SkillOrder))
	for _, skill := range SkillOrder {
		row, ok := bySkill[skill]
		if !ok {
			result = append(result, SkillRating{
				Skill:         string(skill),
				Rating:        rating.StartRating,
				Level:         rating.LevelFor(rating.StartRating),
				AttemptsCount: 0,
			})
			continue
		}
		result = append(result, SkillRating{
			Skill:         string(row.Skill),
			Rating:        row.Rating,
			Level:         row.LevelCode,
			AttemptsCount: row.AttemptsCount,
		})
	}
	return result, nil
}

// nextActivities suggests a few activities per skill, ordered weakest skill
// first so the daily plan starts from the biggest gap.
func (s *Service) nextActivities(
	ctx context.Context,
	profileID int64,
	track store.Track,
	level string,
	skills []SkillRating,
) ([]NextActivity, error) {
	ratingBySkill := make(map[store.Skill]float64, len(skills))
	for _, s := range skills {
		ratingBySkill[store.Skill(s.Skill)] = s.Rating
	}

	combined := make([]NextActivity, 0, len(skills)*int(nextPerSkill))
	for _, skill := range SkillOrder {
		rows, err := s.queries.NextActivities(ctx, store.NextActivitiesParams{
			Skill:     skill,
			LevelCode: level,
			Track:     store.NullTrack{Track: track, Valid: true},
			ProfileID: profileID,
			Rating:    int32(ratingBySkill[skill]),
			Lim:       nextPerSkill,
		})
		if err != nil {
			return nil, fmt.Errorf("next activities for %s: %w", skill, err)
		}
		for _, row := range rows {
			combined = append(combined, NextActivity{
				ID:           row.ID,
				UnitID:       row.UnitID,
				UnitPosition: row.UnitPosition,
				UnitTitle:    row.UnitTitle,
				Type:         string(row.Type),
				Skill:        string(row.Skill),
				Difficulty:   row.Difficulty,
				Position:     row.Position,
				rating:       ratingBySkill[skill],
			})
		}
	}

	sort.SliceStable(combined, func(i, j int) bool {
		if combined[i].rating != combined[j].rating {
			return combined[i].rating < combined[j].rating
		}
		return abs(int(combined[i].Difficulty)-int(combined[i].rating)) <
			abs(int(combined[j].Difficulty)-int(combined[j].rating))
	})
	return combined, nil
}

func minutesForWeek(weekly []WeeklyMinutes, week time.Time) int32 {
	for _, w := range weekly {
		if sameDay(w.WeekStart, week) {
			return w.Minutes
		}
	}
	return 0
}

func location(name string) *time.Location {
	if name == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
