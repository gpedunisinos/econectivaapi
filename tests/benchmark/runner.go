package benchmark

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
)

//go:embed cases.json
var casesData []byte

func LoadCases() ([]Case, error) {
	var cases []Case
	decoder := json.NewDecoder(bytes.NewReader(casesData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cases); err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, c := range cases {
		if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(c.ID) || ids[c.ID] || len(c.Tags) == 0 || len(c.Steps) == 0 || len(c.Steps) > 10 || c.Chat != "global" && c.Chat != "clue" {
			return nil, errors.New("invalid or duplicate benchmark case: " + c.ID)
		}
		ids[c.ID] = true
		for _, step := range c.Steps {
			if strings.TrimSpace(step.Message) == "" || len(step.Expectations.RequiredPatterns)+len(step.Expectations.Calls) == 0 && !step.Expectations.ExpectedLimitError {
				return nil, errors.New("case without meaningful input/expectations: " + c.ID)
			}
			for _, pattern := range append(append([]string{}, step.Expectations.RequiredPatterns...), step.Expectations.ForbiddenPatterns...) {
				if _, err := regexp.Compile("(?is)" + pattern); err != nil {
					return nil, fmt.Errorf("case %s invalid pattern: %w", c.ID, err)
				}
			}
		}
	}
	return cases, nil
}

func SelectCases(cases []Case, ids, tags []string) ([]Case, error) {
	chosen := []Case{}
	seen := map[string]bool{}
	for _, c := range cases {
		idMatch := len(ids) == 0
		tagMatch := len(tags) == 0
		for _, id := range ids {
			if c.ID == id {
				idMatch = true
				seen[id] = true
			}
		}
		for _, tag := range tags {
			for _, actual := range c.Tags {
				if tag == actual {
					tagMatch = true
				}
			}
		}
		if idMatch && tagMatch {
			chosen = append(chosen, c)
		}
	}
	for _, id := range ids {
		if !seen[id] {
			return nil, errors.New("unknown benchmark case: " + id)
		}
	}
	if len(chosen) == 0 {
		return nil, errors.New("no benchmark cases match the selection")
	}
	return chosen, nil
}

type Options struct {
	Progress func(CaseResult)
	Mode     string
	Runs     int
}

type scriptedGemini struct {
	script Script
	turn   int
}

func (s *scriptedGemini) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host != "generativelanguage.googleapis.com" {
		return nil, errors.New("deterministic provider refuses other destinations")
	}
	var body map[string]interface{}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		return nil, err
	}
	var parts []map[string]interface{}
	if s.turn < len(s.script.Calls) {
		call := s.script.Calls[s.turn]
		parts = []map[string]interface{}{{"functionCall": map[string]interface{}{"name": call.Name, "args": call.Arguments, "id": fmt.Sprintf("fixture-%d", s.turn)}, "thoughtSignature": "synthetic-signature"}}
	} else {
		parts = []map[string]interface{}{{"text": s.script.Reply}}
	}
	s.turn++
	payload, _ := json.Marshal(map[string]interface{}{"candidates": []interface{}{map[string]interface{}{"content": map[string]interface{}{"role": "model", "parts": parts}, "finishReason": "STOP"}}, "modelVersion": "gemini-3.8-flash-simulated"})
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(payload)), Request: request}, nil
}

func Run(ctx context.Context, cases []Case, options Options) (Report, error) {
	report := Report{SchemaVersion: 1, Mode: options.Mode, FixtureVersion: FixtureVersion, StartedAt: time.Now().UTC(), RequestedModel: "gemini-3.8-flash", Runs: options.Runs, Results: []CaseResult{}, ByTag: map[string]map[string]int{}}
	hash := sha256.Sum256(casesData)
	report.CaseManifestSHA256 = hex.EncodeToString(hash[:])
	if options.Mode != "deterministic" && options.Mode != "gemini" || options.Runs < 1 || options.Runs > 20 {
		return report, errors.New("invalid benchmark mode or run count")
	}
	if options.Mode == "gemini" && os.Getenv("ONLIFE_BENCHMARK_ALLOW_PROVIDER") != "1" {
		return report, errors.New("real Gemini calls require explicit provider opt-in")
	}
	if options.Mode == "gemini" && strings.TrimSpace(os.Getenv("GENAI_API_KEY")) == "" {
		return report, errors.New("provider credential missing")
	}
	if model := os.Getenv("GEMINI_MODEL"); model != "" && model != "gemini-3.8-flash" {
		return report, errors.New("reference benchmark requires gemini-3.8-flash without fallback")
	}
	previousKey, keySet := os.LookupEnv("GENAI_API_KEY")
	previousModel, modelSet := os.LookupEnv("GEMINI_MODEL")
	defer func() {
		if keySet {
			os.Setenv("GENAI_API_KEY", previousKey)
		} else {
			os.Unsetenv("GENAI_API_KEY")
		}
		if modelSet {
			os.Setenv("GEMINI_MODEL", previousModel)
		} else {
			os.Unsetenv("GEMINI_MODEL")
		}
	}()
	if options.Mode == "deterministic" {
		os.Setenv("GENAI_API_KEY", "synthetic-benchmark-not-a-provider-credential")
	}
	os.Setenv("GEMINI_MODEL", "gemini-3.8-flash")
	if commit, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		report.Commit = strings.TrimSpace(string(commit))
	}
	start := time.Now()
	for _, c := range cases {
		for run := 1; run <= options.Runs; run++ {
			result := CaseResult{ID: c.ID, Tags: c.Tags, Run: run, Passed: true, Steps: []StepResult{}}
			// No database chat writes: history is isolated per run, user and chat identity.
			histories := map[string][]models.ChatMessage{}
			for index, step := range c.Steps {
				actual, history := safeRunStep(ctx, c, step, index, options, histories)
				result.Steps = append(result.Steps, actual)
				if !actual.Passed {
					result.Passed = false
				}
				key := historyKey(c, step)
				histories[key] = history
			}
			if result.Passed {
				report.Passed++
			} else {
				report.Failed++
			}
			for _, tag := range c.Tags {
				if report.ByTag[tag] == nil {
					report.ByTag[tag] = map[string]int{"passed": 0, "failed": 0}
				}
				if result.Passed {
					report.ByTag[tag]["passed"]++
				} else {
					report.ByTag[tag]["failed"]++
				}
			}
			report.Results = append(report.Results, result)
			if options.Progress != nil {
				options.Progress(result)
			}
		}
	}
	report.DurationMS = time.Since(start).Milliseconds()
	return report, nil
}

func historyKey(c Case, s Step) string {
	chat := s.Chat
	if chat == "" {
		chat = c.Chat
	}
	user := s.User
	if user == "" {
		user = "alice"
	}
	if chat == "clue" {
		return fmt.Sprintf("%s:%s:%d", user, chat, s.ClueID)
	}
	return user + ":" + chat
}
func runStep(ctx context.Context, c Case, step Step, index int, options Options, histories map[string][]models.ChatMessage) (StepResult, []models.ChatMessage) {
	chat := step.Chat
	if chat == "" {
		chat = c.Chat
	}
	name := step.User
	if name == "" {
		name = "alice"
	}
	history := append([]models.ChatMessage{}, histories[historyKey(c, step)]...)
	history = append(history, step.History...)
	result := StepResult{Step: index + 1, Chat: chat, User: name, Input: step.Message, Context: step.Screen, Failures: []string{}}
	user, err := FixtureUser(name)
	if err != nil {
		result.Error = "synthetic account unavailable"
		result.Failures = []string{result.Error}
		result.FailureCategory = "integration"
		return result, history
	}
	input := services.ChatCompletionInput{User: &user, ScreenContext: &step.Screen, Message: step.Message, History: history, Template: services.ChatTemplateContext{Name: "ai_messages", SystemInstruction: services.BuildAIMessageSystemInstruction()}}
	if chat == "clue" {
		clue, readings, loadErr := FixtureClue(ctx, step.ClueID)
		if loadErr != nil {
			result.Error = "synthetic clue unavailable"
			result.Failures = []string{result.Error}
			result.FailureCategory = "integration"
			return result, history
		}
		input.PinnedClueID = step.ClueID
		input.Template = services.ChatTemplateContext{Name: "living_clue_chat", SystemInstruction: services.BuildLivingClueSystemInstruction()}
		input.Context = services.BuildLivingClueChatContext(clue, readings)
		// A clue's endpoint has its own server-built context, never a global UI prompt.
		input.ScreenContext = nil
	}
	service := &services.GeminiChatService{ToolFactory: FixtureQueryFactory(step.SourceMode), OnTrace: func(trace services.ChatTrace) { result.Trace = trace }}
	if options.Mode == "deterministic" {
		service.Client = &http.Client{Transport: &scriptedGemini{script: step.Simulation}}
	}
	response, err := service.GenerateReply(ctx, input)
	result.Response = response
	if err != nil {
		result.Error = err.Error()
	}
	result.Failures = Evaluate(step, response, result.Trace, err)
	if options.Mode == "gemini" && err == nil && !strings.HasPrefix(result.Trace.ReturnedModel, "gemini-3.8-flash") {
		result.Failures = append(result.Failures, "modelo retornado não comprova Gemini 3.8 Flash")
	}
	result.Passed = len(result.Failures) == 0
	if !result.Passed {
		result.FailureCategory = "content"
		if err != nil {
			result.FailureCategory = "integration"
			if errors.Is(err, services.ErrGeminiQuota) || errors.Is(err, services.ErrGeminiModel) || errors.Is(err, services.ErrGeminiUnavailable) {
				result.FailureCategory = "provider"
			}
		}
	}
	if err == nil {
		history = append(history, models.ChatMessage{Role: "user", Content: step.Message}, models.ChatMessage{Role: "assistant", Content: response})
	}
	return result, history
}

// An integration bug must produce a failed report row, never a successful or
// truncated benchmark. Do not serialize panic payloads, which can contain data.
func safeRunStep(ctx context.Context, c Case, step Step, index int, options Options, histories map[string][]models.ChatMessage) (result StepResult, history []models.ChatMessage) {
	defer func() {
		if recover() != nil {
			result = StepResult{Step: index + 1, Chat: c.Chat, User: step.User, Input: step.Message, Context: step.Screen, Passed: false, Error: "falha inesperada da integração nesta etapa", Failures: []string{"falha inesperada da integração nesta etapa"}, FailureCategory: "integration"}
			history = histories[historyKey(c, step)]
		}
	}()
	return runStep(ctx, c, step, index, options, histories)
}
