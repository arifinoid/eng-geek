package gate

import (
	"strings"
	"testing"
)

func TestEvaluate(t *testing.T) {
	checkpoint := 0.8

	tests := []struct {
		name       string
		input      Input
		wantPassed bool
		wantReason string
	}{
		{
			name: "all rules met",
			input: Input{
				Completion:        Completion{Total: 10, Done: 8},
				CheckpointOverall: &checkpoint,
				Ratings:           map[string]float64{"listening": 1000, "reading": 1010, "writing": 1000, "speaking": 1200},
			},
			wantPassed: true,
		},
		{
			name: "completion just below the ratio",
			input: Input{
				Completion:        Completion{Total: 10, Done: 7},
				CheckpointOverall: &checkpoint,
				Ratings:           map[string]float64{"listening": 1000, "reading": 1000, "writing": 1000, "speaking": 1000},
			},
			wantReason: "completion 70%",
		},
		{
			name: "no activities in the level",
			input: Input{
				Completion:        Completion{Total: 0, Done: 0},
				CheckpointOverall: &checkpoint,
				Ratings:           map[string]float64{"listening": 1000, "reading": 1000, "writing": 1000, "speaking": 1000},
			},
			wantReason: "completion 0%",
		},
		{
			name: "checkpoint missing",
			input: Input{
				Completion: Completion{Total: 10, Done: 10},
				Ratings:    map[string]float64{"listening": 1000, "reading": 1000, "writing": 1000, "speaking": 1000},
			},
			wantReason: "no checkpoint recorded",
		},
		{
			name: "checkpoint below the threshold",
			input: Input{
				Completion:        Completion{Total: 10, Done: 10},
				CheckpointOverall: ptr(0.69),
				Ratings:           map[string]float64{"listening": 1000, "reading": 1000, "writing": 1000, "speaking": 1000},
			},
			wantReason: "checkpoint overall 0.69",
		},
		{
			name: "one skill below the threshold",
			input: Input{
				Completion:        Completion{Total: 10, Done: 10},
				CheckpointOverall: &checkpoint,
				Ratings:           map[string]float64{"listening": 1000, "reading": 999, "writing": 1000, "speaking": 1000},
			},
			wantReason: "reading rating 999",
		},
		{
			name: "unattempted skill falls back to the start rating",
			input: Input{
				Completion:        Completion{Total: 10, Done: 10},
				CheckpointOverall: &checkpoint,
				Ratings:           map[string]float64{"reading": 1000, "writing": 1000, "speaking": 1000},
			},
			wantReason: "listening rating 800",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.input, 800)
			if got.Passed != tt.wantPassed {
				t.Fatalf("passed = %v, want %v (reasons: %v)", got.Passed, tt.wantPassed, got.Reasons)
			}
			if tt.wantReason == "" {
				if len(got.Reasons) != 0 {
					t.Fatalf("reasons = %v, want none", got.Reasons)
				}
				return
			}
			if !anyContains(got.Reasons, tt.wantReason) {
				t.Fatalf("reasons = %v, want one containing %q", got.Reasons, tt.wantReason)
			}
		})
	}
}

func TestEvaluateReportsTheWeakestSkill(t *testing.T) {
	got := Evaluate(Input{
		Completion:        Completion{Total: 10, Done: 10},
		CheckpointOverall: ptr(0.8),
		Ratings: map[string]float64{
			"listening": 1100, "reading": 900, "writing": 1050, "speaking": 1200,
		},
	}, 800)

	if got.MinSkill != "reading" {
		t.Fatalf("min skill = %q, want reading", got.MinSkill)
	}
	if got.MinRating != 900 {
		t.Fatalf("min rating = %v, want 900", got.MinRating)
	}
}

func ptr(value float64) *float64 { return &value }

func anyContains(values []string, substr string) bool {
	for _, value := range values {
		if strings.Contains(value, substr) {
			return true
		}
	}
	return false
}
