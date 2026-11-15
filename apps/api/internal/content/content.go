// Package content loads and validates A1 course content YAML before it is
// seeded into Postgres. Validation is pure (no DB) so it is easy to test.
package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Unit is one course unit: metadata, vocabulary and its activities.
type Unit struct {
	Level      string
	Week       int
	Position   int
	Track      *string
	Title      string
	Vocab      []Vocab
	Activities []Activity
}

// unitFile mirrors the YAML layout: a `unit` mapping plus sibling `vocab` and
// `activities` lists.
type unitFile struct {
	Unit struct {
		Level    string  `yaml:"level"`
		Week     int     `yaml:"week"`
		Position int     `yaml:"position"`
		Track    *string `yaml:"track"`
		Title    string  `yaml:"title"`
	} `yaml:"unit"`
	Vocab      []Vocab    `yaml:"vocab"`
	Activities []Activity `yaml:"activities"`
}

// Vocab is one vocabulary item.
type Vocab struct {
	Term    string  `yaml:"term" json:"term"`
	Meaning string  `yaml:"meaning" json:"meaning"`
	Example *string `yaml:"example" json:"example"`
}

// Activity is one activity inside a unit.
type Activity struct {
	Type       string          `yaml:"type" json:"type"`
	Skill      string          `yaml:"skill" json:"skill"`
	Difficulty int             `yaml:"difficulty" json:"difficulty"`
	Content    ActivityContent `yaml:"content" json:"content"`
}

// ActivityContent is the JSONB payload stored in activities.content. The shape
// depends on the activity type; fields that do not apply stay empty.
//
// The `questions` key is polymorphic: quiz/vocab/listening use structured
// Question values, while speaking "answer" mode uses plain prompt strings.
type ActivityContent struct {
	Items            []Vocab
	Questions        []Question
	Prompts          []string
	Passage          *string
	Script           *string
	Audio            *string
	Prompt           *string
	MinWords         *int
	Rubric           []string
	ModelAnswer      *string
	Mode             *string
	Lines            []string
	ExpectedKeywords []string
}

// UnmarshalYAML splits the polymorphic `questions` key into structured
// questions or plain prompts.
func (c *ActivityContent) UnmarshalYAML(node *yaml.Node) error {
	var raw struct {
		Items            []Vocab   `yaml:"items"`
		Questions        yaml.Node `yaml:"questions"`
		Passage          *string   `yaml:"passage"`
		Script           *string   `yaml:"script"`
		Audio            *string   `yaml:"audio"`
		Prompt           *string   `yaml:"prompt"`
		MinWords         *int      `yaml:"min_words"`
		Rubric           []string  `yaml:"rubric"`
		ModelAnswer      *string   `yaml:"model_answer"`
		Mode             *string   `yaml:"mode"`
		Lines            []string  `yaml:"lines"`
		ExpectedKeywords []string  `yaml:"expected_keywords"`
	}
	if err := node.Decode(&raw); err != nil {
		return err
	}

	c.Items = raw.Items
	c.Passage = raw.Passage
	c.Script = raw.Script
	c.Audio = raw.Audio
	c.Prompt = raw.Prompt
	c.MinWords = raw.MinWords
	c.Rubric = raw.Rubric
	c.ModelAnswer = raw.ModelAnswer
	c.Mode = raw.Mode
	c.Lines = raw.Lines
	c.ExpectedKeywords = raw.ExpectedKeywords

	switch {
	case raw.Questions.Kind == 0:
		// No questions key.
	case raw.Questions.Kind == yaml.SequenceNode &&
		len(raw.Questions.Content) > 0 &&
		raw.Questions.Content[0].Kind == yaml.ScalarNode:
		var prompts []string
		if err := raw.Questions.Decode(&prompts); err != nil {
			return err
		}
		c.Prompts = prompts
	default:
		var questions []Question
		if err := raw.Questions.Decode(&questions); err != nil {
			return err
		}
		c.Questions = questions
	}
	return nil
}

// MarshalJSON writes only the fields that apply, keeping the documented shape
// for each activity type.
func (c ActivityContent) MarshalJSON() ([]byte, error) {
	out := make(map[string]any, 10)
	if len(c.Items) > 0 {
		out["items"] = c.Items
	}
	if len(c.Questions) > 0 {
		out["questions"] = c.Questions
	} else if len(c.Prompts) > 0 {
		out["questions"] = c.Prompts
	}
	if c.Passage != nil {
		out["passage"] = c.Passage
	}
	if c.Script != nil {
		out["script"] = c.Script
	}
	if c.Audio != nil {
		out["audio"] = c.Audio
	}
	if c.Prompt != nil {
		out["prompt"] = c.Prompt
	}
	if c.MinWords != nil {
		out["min_words"] = c.MinWords
	}
	if len(c.Rubric) > 0 {
		out["rubric"] = c.Rubric
	}
	if c.ModelAnswer != nil {
		out["model_answer"] = c.ModelAnswer
	}
	if c.Mode != nil {
		out["mode"] = c.Mode
	}
	if len(c.Lines) > 0 {
		out["lines"] = c.Lines
	}
	if len(c.ExpectedKeywords) > 0 {
		out["expected_keywords"] = c.ExpectedKeywords
	}
	return json.Marshal(out)
}

// Question is a multiple-choice question.
type Question struct {
	Prompt  string   `yaml:"prompt" json:"prompt"`
	Options []string `yaml:"options" json:"options"`
	Answer  int      `yaml:"answer" json:"answer"`
	Explain *string  `yaml:"explain" json:"explain,omitempty"`
}

// JSON encodes the activity content for the JSONB column.
func (c ActivityContent) JSON() ([]byte, error) {
	return json.Marshal(c)
}

var (
	validTypes  = map[string]bool{"quiz": true, "vocab": true, "writing": true, "speaking": true, "listening": true}
	validSkills = map[string]bool{"listening": true, "reading": true, "writing": true, "speaking": true}
	validTracks = map[string]bool{"engineer": true, "nurse": true}
	allSkills   = []string{"listening", "reading", "writing", "speaking"}
)

// activityTemplate is the fixed 9-activity order per unit (see docs/domain.md).
var activityTemplate = []struct {
	Type   string
	Skill  string
	Offset int
}{
	{"vocab", "reading", 0},
	{"quiz", "reading", 0},
	{"quiz", "reading", 10},
	{"listening", "listening", -10},
	{"listening", "listening", 10},
	{"quiz", "reading", 0},
	{"writing", "writing", 20},
	{"speaking", "speaking", -10},
	{"speaking", "speaking", 20},
}

// LoadUnits reads every *.yaml file in dir, validates each unit, and returns
// them ordered by position. Filename must start with the unit position.
func LoadUnits(dir string) ([]Unit, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read content dir: %w", err)
	}

	var units []Unit
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}

		var doc unitFile
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&doc); err != nil {
			return nil, fmt.Errorf("%s: parse: %w", entry.Name(), err)
		}
		unit := Unit{
			Level:      doc.Unit.Level,
			Week:       doc.Unit.Week,
			Position:   doc.Unit.Position,
			Track:      doc.Unit.Track,
			Title:      doc.Unit.Title,
			Vocab:      doc.Vocab,
			Activities: doc.Activities,
		}
		if err := ValidateUnit(unit); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		if err := checkFilename(entry.Name(), unit); err != nil {
			return nil, err
		}
		units = append(units, unit)
	}

	if len(units) == 0 {
		return nil, fmt.Errorf("no content files in %s", dir)
	}

	sort.Slice(units, func(i, j int) bool { return units[i].Position < units[j].Position })
	return units, nil
}

func checkFilename(name string, unit Unit) error {
	prefix, _, ok := strings.Cut(name, "-")
	if !ok {
		return fmt.Errorf("%s: filename must start with the unit position", name)
	}
	n, err := strconv.Atoi(prefix)
	if err != nil || n != unit.Position {
		return fmt.Errorf("%s: filename position %q does not match position %d", name, prefix, unit.Position)
	}
	return nil
}

// ValidateUnit checks the domain rules that must hold before content is
// written: track, title, vocabulary, the fixed activity template, difficulty
// formula, and question answers.
func ValidateUnit(unit Unit) error {
	if unit.Level == "" {
		return fmt.Errorf("level is required")
	}
	if unit.Track != nil && !validTracks[*unit.Track] {
		return fmt.Errorf("track %q is not engineer/nurse", *unit.Track)
	}
	if unit.Position < 1 {
		return fmt.Errorf("position must be >= 1, got %d", unit.Position)
	}
	if unit.Week < 1 {
		return fmt.Errorf("week must be >= 1, got %d", unit.Week)
	}
	if strings.TrimSpace(unit.Title) == "" {
		return fmt.Errorf("title is required")
	}

	if len(unit.Vocab) == 0 {
		return fmt.Errorf("vocab must not be empty")
	}
	for i, v := range unit.Vocab {
		if strings.TrimSpace(v.Term) == "" || strings.TrimSpace(v.Meaning) == "" {
			return fmt.Errorf("vocab[%d]: term and meaning are required", i)
		}
	}

	if len(unit.Activities) != len(activityTemplate) {
		return fmt.Errorf("activities: got %d, want %d", len(unit.Activities), len(activityTemplate))
	}

	skills := map[string]bool{}
	for i, activity := range unit.Activities {
		want := activityTemplate[i]
		if activity.Type != want.Type {
			return fmt.Errorf("activities[%d]: type %q, want %q", i, activity.Type, want.Type)
		}
		if activity.Skill != want.Skill {
			return fmt.Errorf("activities[%d]: skill %q, want %q", i, activity.Skill, want.Skill)
		}
		if !validTypes[activity.Type] || !validSkills[activity.Skill] {
			return fmt.Errorf("activities[%d]: unknown type/skill", i)
		}
		skills[activity.Skill] = true

		base := 840 + (unit.Week-1)*10
		wantDifficulty := base + want.Offset
		if activity.Difficulty != wantDifficulty {
			return fmt.Errorf("activities[%d]: difficulty %d, want %d", i, activity.Difficulty, wantDifficulty)
		}
		if err := validateActivityContent(activity); err != nil {
			return fmt.Errorf("activities[%d]: %w", i, err)
		}
	}

	for _, skill := range allSkills {
		if !skills[skill] {
			return fmt.Errorf("unit must cover all four skills, missing %s", skill)
		}
	}

	return nil
}

func validateActivityContent(activity Activity) error {
	switch activity.Type {
	case "vocab":
		// Items are optional: the quiz usually refers to the unit's vocab list.
		for i, item := range activity.Content.Items {
			if strings.TrimSpace(item.Term) == "" || strings.TrimSpace(item.Meaning) == "" {
				return fmt.Errorf("items[%d]: term and meaning are required", i)
			}
		}
		return validateQuestions(activity.Content.Questions)
	case "quiz":
		if err := validateQuestions(activity.Content.Questions); err != nil {
			return err
		}
		if activity.Content.Passage == nil && len(activity.Content.Questions) == 0 {
			return fmt.Errorf("quiz content needs questions or a passage")
		}
		return nil
	case "listening":
		if activity.Content.Script == nil || strings.TrimSpace(*activity.Content.Script) == "" {
			return fmt.Errorf("listening content needs a script")
		}
		return validateQuestions(activity.Content.Questions)
	case "writing":
		if activity.Content.Prompt == nil || strings.TrimSpace(*activity.Content.Prompt) == "" {
			return fmt.Errorf("writing content needs a prompt")
		}
		if activity.Content.MinWords == nil || *activity.Content.MinWords <= 0 {
			return fmt.Errorf("writing content needs min_words > 0")
		}
		return nil
	case "speaking":
		mode := ""
		if activity.Content.Mode != nil {
			mode = *activity.Content.Mode
		}
		switch mode {
		case "shadow":
			if len(activity.Content.Lines) == 0 {
				return fmt.Errorf("shadow speaking needs lines")
			}
		case "answer":
			if len(activity.Content.Prompts) == 0 {
				return fmt.Errorf("answer speaking needs questions")
			}
		default:
			return fmt.Errorf("speaking mode %q is not shadow/answer", mode)
		}
		return nil
	default:
		return fmt.Errorf("unknown activity type %q", activity.Type)
	}
}

func validateQuestions(questions []Question) error {
	if len(questions) == 0 {
		return fmt.Errorf("questions must not be empty")
	}
	for i, q := range questions {
		if strings.TrimSpace(q.Prompt) == "" {
			return fmt.Errorf("questions[%d]: prompt is required", i)
		}
		if len(q.Options) < 2 {
			return fmt.Errorf("questions[%d]: need at least 2 options", i)
		}
		if q.Answer < 0 || q.Answer >= len(q.Options) {
			return fmt.Errorf("questions[%d]: answer %d out of range [0,%d]", i, q.Answer, len(q.Options)-1)
		}
	}
	return nil
}
