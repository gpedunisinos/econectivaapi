// Package benchmark runs synthetic education conversations through production
// prompts, Gemini orchestration and the official MCP client/server protocol.
package benchmark

import (
	"encoding/json"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
)

const FixtureVersion = "onlife-usp-synthetic-2"

type CallExpectation struct {
	ResultPatterns []string               `json:"result_patterns,omitempty"`
	Name           string                 `json:"name"`
	Arguments      map[string]interface{} `json:"arguments"`
	MustFail       bool                   `json:"must_fail,omitempty"`
}
type Expectations struct {
	NoCalls            bool              `json:"no_calls,omitempty"`
	RequiredPatterns   []string          `json:"required_patterns"`
	ForbiddenPatterns  []string          `json:"forbidden_patterns"`
	Calls              []CallExpectation `json:"calls"`
	ForbiddenCalls     []string          `json:"forbidden_calls,omitempty"`
	MaximumCalls       int               `json:"maximum_calls,omitempty"`
	ExpectedLimitError bool              `json:"expected_limit_error,omitempty"`
}
type Script struct {
	Calls []CallExpectation `json:"calls"`
	Reply string            `json:"reply"`
}
type Step struct {
	Message      string                 `json:"message"`
	Screen       models.AIScreenContext `json:"screen"`
	Chat         string                 `json:"chat,omitempty"`
	User         string                 `json:"user,omitempty"`
	ClueID       uint                   `json:"clue_id,omitempty"`
	SourceMode   string                 `json:"source_mode,omitempty"`
	History      []models.ChatMessage   `json:"history,omitempty"`
	Expectations Expectations           `json:"expectations"`
	Simulation   Script                 `json:"simulation"`
}
type Case struct {
	ID          string   `json:"id"`
	Tags        []string `json:"tags"`
	Chat        string   `json:"chat"`
	Description string   `json:"description"`
	Steps       []Step   `json:"steps"`
}
type StepResult struct {
	Step            int                    `json:"step"`
	Chat            string                 `json:"chat"`
	User            string                 `json:"user"`
	Input           string                 `json:"input"`
	Context         models.AIScreenContext `json:"context"`
	Response        string                 `json:"response"`
	Trace           services.ChatTrace     `json:"trace"`
	Passed          bool                   `json:"passed"`
	Failures        []string               `json:"failures"`
	Error           string                 `json:"error,omitempty"`
	FailureCategory string                 `json:"failure_category,omitempty"`
}
type CaseResult struct {
	ID     string       `json:"id"`
	Tags   []string     `json:"tags"`
	Run    int          `json:"run"`
	Passed bool         `json:"passed"`
	Steps  []StepResult `json:"steps"`
}
type Report struct {
	CaseManifestSHA256 string                    `json:"case_manifest_sha256"`
	SchemaVersion      int                       `json:"schema_version"`
	Mode               string                    `json:"mode"`
	FixtureVersion     string                    `json:"fixture_version"`
	Commit             string                    `json:"commit"`
	StartedAt          time.Time                 `json:"started_at"`
	DurationMS         int64                     `json:"duration_ms"`
	RequestedModel     string                    `json:"requested_model"`
	Runs               int                       `json:"runs"`
	Results            []CaseResult              `json:"results"`
	Passed             int                       `json:"passed"`
	Failed             int                       `json:"failed"`
	ByTag              map[string]map[string]int `json:"by_tag"`
}

func sameArguments(actual json.RawMessage, expected map[string]interface{}) bool {
	var value map[string]interface{}
	if json.Unmarshal(actual, &value) != nil {
		return false
	}
	for key, want := range expected {
		a, _ := json.Marshal(value[key])
		b, _ := json.Marshal(want)
		if string(a) != string(b) {
			return false
		}
	}
	return true
}
