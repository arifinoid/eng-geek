package content

import (
	"testing"
)

// cloneUnit deep-copies the slices so a mutation in one case cannot leak into
// the shared fixture or the next case.
func cloneUnit(unit Unit) Unit {
	clone := unit
	clone.Vocab = append([]Vocab(nil), unit.Vocab...)
	clone.Activities = make([]Activity, len(unit.Activities))
	for i, activity := range unit.Activities {
		content := activity.Content
		content.Items = append([]Vocab(nil), activity.Content.Items...)
		content.Questions = append([]Question(nil), activity.Content.Questions...)
		content.Prompts = append([]string(nil), activity.Content.Prompts...)
		content.Rubric = append([]string(nil), activity.Content.Rubric...)
		content.Lines = append([]string(nil), activity.Content.Lines...)
		content.ExpectedKeywords = append([]string(nil), activity.Content.ExpectedKeywords...)
		activity.Content = content
		clone.Activities[i] = activity
	}
	return clone
}

func TestLoadUnits(t *testing.T) {
	units, err := LoadUnits("../../content/a1")
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("got %d units, want 1", len(units))
	}

	unit := units[0]
	if unit.Level != "A1" {
		t.Errorf("level = %q, want A1", unit.Level)
	}
	if unit.Week != 2 || unit.Position != 2 {
		t.Errorf("week/position = %d/%d, want 2/2", unit.Week, unit.Position)
	}
	if unit.Track != nil {
		t.Errorf("track = %v, want nil (shared)", *unit.Track)
	}
	if len(unit.Vocab) != 14 {
		t.Errorf("vocab = %d, want 14", len(unit.Vocab))
	}
	if len(unit.Activities) != 9 {
		t.Errorf("activities = %d, want 9", len(unit.Activities))
	}
}

func TestValidateUnitAcceptsFixture(t *testing.T) {
	units, err := LoadUnits("../../content/a1")
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	for _, unit := range units {
		if err := ValidateUnit(unit); err != nil {
			t.Errorf("ValidateUnit(%s): %v", unit.Title, err)
		}
	}
}

func TestValidateUnitRejectsBadContent(t *testing.T) {
	units, err := LoadUnits("../../content/a1")
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	base := units[0]

	track := "doctor"
	cases := []struct {
		name   string
		mutate func(*Unit)
	}{
		{"wrong difficulty", func(u *Unit) { u.Activities[0].Difficulty++ }},
		{"answer out of range", func(u *Unit) { u.Activities[1].Content.Questions[0].Answer = 99 }},
		{"wrong activity type", func(u *Unit) { u.Activities[0].Type = "quiz" }},
		{"wrong activity skill", func(u *Unit) { u.Activities[0].Skill = "listening" }},
		{"missing activity", func(u *Unit) { u.Activities = u.Activities[:8] }},
		{"unknown track", func(u *Unit) { u.Track = &track }},
		{"zero position", func(u *Unit) { u.Position = 0 }},
		{"zero week", func(u *Unit) { u.Week = 0 }},
		{"empty title", func(u *Unit) { u.Title = " " }},
		{"empty vocab", func(u *Unit) { u.Vocab = nil }},
		{"empty term", func(u *Unit) { u.Vocab[0].Term = "" }},
		{"vocab without questions", func(u *Unit) { u.Activities[0].Content.Questions = nil }},
		{"listening without script", func(u *Unit) { u.Activities[3].Content.Script = nil }},
		{"writing without prompt", func(u *Unit) { u.Activities[6].Content.Prompt = nil }},
		{"speaking unknown mode", func(u *Unit) { mode := "chat"; u.Activities[7].Content.Mode = &mode }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unit := cloneUnit(base)
			tc.mutate(&unit)
			if err := ValidateUnit(unit); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestFilenameMustMatchPosition(t *testing.T) {
	if err := checkFilename("02-greetings.yaml", Unit{Position: 2}); err != nil {
		t.Errorf("matching filename: %v", err)
	}
	if err := checkFilename("99-greetings.yaml", Unit{Position: 2}); err == nil {
		t.Error("expected filename/position mismatch error")
	}
}

func TestActivityContentJSON(t *testing.T) {
	units, err := LoadUnits("../../content/a1")
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	raw, err := units[0].Activities[0].Content.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected non-empty JSON")
	}
}
