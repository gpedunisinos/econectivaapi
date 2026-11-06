package services

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/models"
)

type sourceTransport func(*http.Request) (*http.Response, error)

func (f sourceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSourceExtractionDeduplicatesAndHidesCredentialURLs(t *testing.T) {
	ops := []map[string]interface{}{
		{"insert": "https://source.example.org/article#x\n"},
		{"insert": "Artigo", "attributes": map[string]interface{}{"link": "https://source.example.org/article"}},
		{"insert": " https://source.example.org/private?token=secret-value\n"},
		{"insert": map[string]interface{}{"onlife": `{"v":1,"provider":"padlet","url":"https://padlet.com/embed/abcdefghijklmnop"}`}},
		{"insert": "\n"},
		{"insert": map[string]interface{}{"onlife": `{"v":1,"provider":"agent","url":"/space/agents/503"}`}},
		{"insert": map[string]interface{}{"onlife": `{"v":1,"provider":"node","url":"/nodes/graph?focus=503"}`}},
	}
	raw, _ := json.Marshal(ops)
	sources := ClueSources(&models.LivingClue{DescriptionDelta: string(raw)})
	if len(sources) != 3 || len(sources[0].Origins) != 2 {
		t.Fatalf("unexpected sources: %#v", sources)
	}
	if sources[1].Consultable || sources[1].URL != "" {
		t.Fatal("credential URL leaked")
	}
	if sources[2].Provider != "padlet" {
		t.Fatal("embed was not identified")
	}
}

func TestPublicSourcePolicyBlocksSensitiveTargetsAndRedirects(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1/x", "http://169.254.169.254/latest/meta-data", "http://10.0.0.1", "http://192.168.1.1", "http://[::1]", "http://[::ffff:127.0.0.1]", "https://user:password@example.org", "https://example.org/?api_key=secret", "http://localhost", "http://LOCALHOST", "https://example.org:8080", "file:///etc/passwd"} {
		if _, err := SafeSourceURL(raw); err == nil {
			t.Fatalf("unsafe source accepted: %s", raw)
		}
	}
	for _, ip := range []string{"172.16.0.1", "100.64.0.1", "198.18.0.1", "203.0.113.1", "2001:db8::1", "64:ff9b::7f00:1", "fc00::1"} {
		if publicIP(net.ParseIP(ip)) {
			t.Fatalf("unsafe resolution accepted: %s", ip)
		}
	}
	client := newPublicSourceClient()
	u, _ := url.Parse("http://169.254.169.254/")
	if client.CheckRedirect(&http.Request{URL: u}, []*http.Request{{}}) == nil {
		t.Fatal("metadata redirect accepted")
	}
}

func TestSourceFetchUsesTextAndCacheWithoutScripts(t *testing.T) {
	calls := 0
	fetcher := NewSourceFetcherWithTransport(sourceTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/html"}}, Body: io.NopCloser(strings.NewReader(`<html><script>do_not_execute_or_quote()</script><body><h1>Observação de teste</h1><p>A área possui 42 árvores em uma amostra sintética.</p></body></html>`)), Request: req}, nil
	}))
	result := fetcher.Fetch(context.Background(), "https://source.example.org/article")
	if result.Status != "ok" || !strings.Contains(result.Text, "42 árvores") || strings.Contains(result.Text, "do_not_execute") {
		t.Fatalf("unexpected extraction: %#v", result)
	}
	_ = fetcher.Fetch(context.Background(), "https://source.example.org/article")
	if calls != 1 {
		t.Fatal("source cache was not used")
	}
	blocked := fetcher.Fetch(context.Background(), "http://127.0.0.1/private")
	if blocked.Status != "blocked" || calls != 1 {
		t.Fatal("blocked destination reached transport")
	}
}

func TestMeetingReferenceIsInteractiveAndDoesNotImplyARecordOfTheCall(t *testing.T) {
	t.Setenv("JITSI_PUBLIC_URL", "https://meet.spotec.app")
	sources := ClueSources(&models.LivingClue{Description: "Link associado https://meet.spotec.app/encontro-ficticio"})
	if len(sources) != 1 || sources[0].Provider != "jitsi" {
		t.Fatalf("meeting must remain an interactive reference: %#v", sources)
	}
}
