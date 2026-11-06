package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/mcpserver"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
)

type modelTransport func(*http.Request) (*http.Response, error)

func (f modelTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type educationFixture struct{ calls int }

func (f *educationFixture) Execute(_ context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
	f.calls++
	if name != "pistas_pesquisar" {
		return nil, errors.New("consulta não autorizada na fixture")
	}
	return map[string]interface{}{"records": []interface{}{map[string]interface{}{"id": 42, "name": "Amostra de água", "link": "#/space/research/42"}}}, nil
}
func modelResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}
}

func TestGeminiExecutesOfficialMCPAndPreservesThoughtSignatures(t *testing.T) {
	t.Setenv("GENAI_API_KEY", "synthetic-secret-never-in-url")
	t.Setenv("GEMINI_MODEL", "gemini-3.8-flash")
	requests := 0
	backend := &educationFixture{}
	service := services.NewGeminiChatService(&http.Client{Transport: modelTransport(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.RawQuery != "" || req.Header.Get("x-goog-api-key") != "synthetic-secret-never-in-url" {
			t.Fatal("credential was not isolated in header")
		}
		var body map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if requests == 1 {
			if len(body["tools"].([]interface{})) != 1 {
				t.Fatal("no tool declarations")
			}
			return modelResponse(req, 200, `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"pistas_pesquisar","args":{"text":"água","limit":3}},"thoughtSignature":"opaque-signature-preserve"}]}}],"modelVersion":"gemini-3.8-flash","usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":3,"totalTokenCount":13}}`), nil
		}
		contents := body["contents"].([]interface{})
		model := contents[len(contents)-2].(map[string]interface{})
		part := model["parts"].([]interface{})[0].(map[string]interface{})
		if part["thoughtSignature"] != "opaque-signature-preserve" {
			t.Fatal("thought signature discarded")
		}
		response := contents[len(contents)-1].(map[string]interface{})["parts"].([]interface{})[0].(map[string]interface{})["functionResponse"].(map[string]interface{})
		if response["name"] != "pistas_pesquisar" || response["response"] == nil {
			t.Fatal("MCP response not returned to model")
		}
		return modelResponse(req, 200, `{"candidates":[{"content":{"role":"model","parts":[{"text":"Encontrei a pista "},{"text":"[Amostra de água](#/space/research/42)."}]}}],"modelVersion":"gemini-3.8-flash"}`), nil
	})})
	service.ToolFactory = func(ctx context.Context, _ models.User, _ uint) (services.QuerySession, error) {
		return mcpserver.ConnectBackend(ctx, backend)
	}
	var trace services.ChatTrace
	service.OnTrace = func(value services.ChatTrace) { trace = value }
	reply, err := service.GenerateReply(context.Background(), services.ChatCompletionInput{User: &models.User{DefaultModel: models.DefaultModel{ID: 1}}, Template: services.ChatTemplateContext{SystemInstruction: services.BuildAIMessageSystemInstruction()}, Message: "Pesquise pistas sobre água."})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply, "#/space/research/42") || backend.calls != 1 || requests != 2 {
		t.Fatalf("not a complete tool cycle: reply=%q backend=%d requests=%d", reply, backend.calls, requests)
	}
	if trace.RequestedModel != "gemini-3.8-flash" || trace.ReturnedModel != "gemini-3.8-flash" || len(trace.Calls) != 1 {
		t.Fatalf("incomplete trace: %#v", trace)
	}
}

func TestGeminiFailuresAreFailuresWithoutSecretEchoOrFallback(t *testing.T) {
	t.Setenv("GENAI_API_KEY", "synthetic-sensitive-value")
	t.Setenv("GEMINI_MODEL", "gemini-3.8-flash")
	for _, status := range []int{401, 404, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			requests := 0
			s := services.NewGeminiChatService(&http.Client{Transport: modelTransport(func(req *http.Request) (*http.Response, error) {
				requests++
				return modelResponse(req, status, `{"error":{"message":"synthetic-sensitive-value"}}`), nil
			})})
			reply, err := s.GenerateReply(context.Background(), services.ChatCompletionInput{Message: "Olá"})
			if err == nil || reply != "" || strings.Contains(err.Error(), "synthetic-sensitive-value") || requests != 1 {
				t.Fatalf("failure not handled safely: %q %v", reply, err)
			}
		})
	}
}

func TestGeminiRepeatedQueriesStopWithoutHanging(t *testing.T) {
	t.Setenv("GENAI_API_KEY", "synthetic-secret")
	backend := &educationFixture{}
	s := services.NewGeminiChatService(&http.Client{Transport: modelTransport(func(req *http.Request) (*http.Response, error) {
		return modelResponse(req, 200, `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"pistas_pesquisar","args":{"text":"água"}}}]}}]}`), nil
	})})
	s.ToolFactory = func(ctx context.Context, _ models.User, _ uint) (services.QuerySession, error) {
		return mcpserver.ConnectBackend(ctx, backend)
	}
	_, err := s.GenerateReply(context.Background(), services.ChatCompletionInput{User: &models.User{}, Message: "Pesquisar"})
	if !errors.Is(err, services.ErrQueryLimit) || backend.calls > 2 {
		t.Fatalf("query limits failed: err=%v calls=%d", err, backend.calls)
	}
}
