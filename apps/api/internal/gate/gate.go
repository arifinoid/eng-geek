// Package gate decides whether a profile may move from one CEFR level to the
// next. The rules are pure (no DB, no HTTP) so they are easy to test.
package gate

import (
	"fmt"
	"math"
)

// Thresholds for the level gate (see docs/domain.md).
const (
	CompletionRatio   = 0.8
	CheckpointOverall = 0.7
	MinSkillRating    = 1000.0
)

// Skills are the four skills every level must cover.
var Skills = []string{"listening", "reading", "writing", "speaking"}

// Completion is how much of a level's activities are done.
type Completion struct {
	Total int
	Done  int
}

// Input is everything the gate needs. Ratings for skills the profile has never
// attempted fall back to startRating.
type Input struct {
	Completion        Completion
	CheckpointOverall *float64
	Ratings           map[string]float64
}

// Result is the gate outcome plus the numbers behind it, so callers can show
// progress without recomputing anything.
type Result struct {
	Passed            bool               `json:"passed"`
	Reasons           []string           `json:"reasons"`
	CompletionTotal   int                `json:"completion_total"`
	CompletionDone    int                `json:"completion_done"`
	CompletionRatio   float64            `json:"completion_ratio"`
	CheckpointOverall *float64           `json:"checkpoint_overall"`
	MinSkill          string             `json:"min_skill"`
	MinRating         float64            `json:"min_rating"`
	Ratings           map[string]float64 `json:"ratings"`
}

// Evaluate applies the three gate rules: at least CompletionRatio of the level's
// activities done, a checkpoint with overall >= CheckpointOverall, and every
// skill rating >= MinSkillRating.
func Evaluate(in Input, startRating float64) Result {
	ratings := make(map[string]float64, len(Skills))
	minSkill := ""
	minRating := math.Inf(1)
	for _, skill := range Skills {
		value, ok := in.Ratings[skill]
		if !ok {
			value = startRating
		}
		ratings[skill] = value
		if value < minRating {
			minSkill, minRating = skill, value
		}
	}

	ratio := 0.0
	if in.Completion.Total > 0 {
		ratio = float64(in.Completion.Done) / float64(in.Completion.Total)
	}

	reasons := make([]string, 0, 3)
	if ratio < CompletionRatio {
		reasons = append(reasons, fmt.Sprintf(
			"completion %.0f%% is below %.0f%% (%d/%d activities)",
			ratio*100, CompletionRatio*100, in.Completion.Done, in.Completion.Total,
		))
	}
	switch {
	case in.CheckpointOverall == nil:
		reasons = append(reasons, "no checkpoint recorded")
	case *in.CheckpointOverall < CheckpointOverall:
		reasons = append(reasons, fmt.Sprintf(
			"checkpoint overall %.2f is below %.2f",
			*in.CheckpointOverall, CheckpointOverall,
		))
	}
	for _, skill := range Skills {
		if ratings[skill] < MinSkillRating {
			reasons = append(reasons, fmt.Sprintf(
				"%s rating %.1f is below %.0f", skill, ratings[skill], MinSkillRating,
			))
		}
	}

	return Result{
		Passed:            len(reasons) == 0,
		Reasons:           reasons,
		CompletionTotal:   in.Completion.Total,
		CompletionDone:    in.Completion.Done,
		CompletionRatio:   ratio,
		CheckpointOverall: in.CheckpointOverall,
		MinSkill:          minSkill,
		MinRating:         minRating,
		Ratings:           ratings,
	}
}
