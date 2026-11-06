package services

import (
	"encoding/json"
	"strings"
	"testing"
)

func integrationDelta(provider, address string) string {
	block, _ := json.Marshal(ClueIntegration{Version: 1, Provider: provider, URL: address})
	delta, _ := json.Marshal([]interface{}{map[string]interface{}{"insert": "Uma pista\n"}, map[string]interface{}{"insert": map[string]interface{}{"onlife": string(block)}}, map[string]interface{}{"insert": "\n"}})
	return string(delta)
}

func TestClueIntegrationContracts(t *testing.T) {
	cases := []struct {
		provider, address string
		valid             bool
	}{
		{"node", "/nodes/graph?focus=503", true},
		{"node", "/nodes/graph?focus=0", false},
		{"node", "/nodes/graph?focus=-1", false},
		{"node", "/nodes/graph?focus=1&extra=x", false},
		{"node", "/nodes/graph?focus=1\n", false},
		{"node", "https://evil.test/nodes/graph?focus=1", false},
		{"node", "javascript:alert(1)", false},
		{"agent", "/space/agents/503", true},
		{"agent", "/chat_templates/chat?id=503", true},
		{"agent", "/chat_templates/chat?id=0", false},
		{"agent", "/chat_templates/chat?id=503&token=x", false},
		{"agent", "/space/agents/0", false},
		{"agent", "/space/agents/-1", false},
		{"agent", "/space/agents/1?token=x", false},
		{"agent", "/space/agents/1/messages", false},
		{"agent", "/space/agents/1\n", false},
		{"agent", "https://evil.test/space/agents/1", false},
		{"agent", "javascript:alert(1)", false},
		{"genially", "https://view.genially.com/60e5417ee87d780d572d8c48/interactive-content", true},
		{"padlet", "https://padlet.com/embed/abcdefghijklmnop", true},
		{"padlet", "https://padlet.com/example/mural-de-pistas-abcd1234", true},
		{"framevr", "https://framevr.io/example-room", true},
		{"overte", "hifi://example-world/1,2,3", true},
		{"vircadia", "hifi://world.example.org", true},
		{"genially", "https://view.genially.com.evil.test/60e5417ee87d780d572d8c48", false},
		{"padlet", "javascript:alert(1)", false},
		{"framevr", "http://framevr.io/room", false},
		{"framevr", "https://user:password@framevr.io/room", false},
		{"framevr", "https://framevr.io:9000/room", false},
		{"padlet", "https://padlet.com/api/users", false},
		{"overte", "file:///tmp/world.json", false},
		{"vircadia", "https://random.example.org", false},
		{"overte", "hifi://127.0.0.1", false},
	}
	for _, c := range cases {
		t.Run(c.provider+"/"+c.address, func(t *testing.T) {
			if got := ValidateQuillDelta(integrationDelta(c.provider, c.address)) == nil; got != c.valid {
				t.Fatalf("valid=%v, want %v", got, c.valid)
			}
		})
	}
}

func TestMixedDocumentPreservesOpaqueBlocksAndSearchableText(t *testing.T) {
	genially, _ := json.Marshal(ClueIntegration{Version: 1, Provider: "genially", URL: "https://view.genially.com/60e5417ee87d780d572d8c48"})
	ops := []map[string]interface{}{
		{"insert": "Texto inicial\n"},
		{"insert": map[string]interface{}{"video": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}},
		{"insert": "\n"},
		{"insert": map[string]interface{}{"onlife": string(genially)}},
		{"insert": "\n"},
		{"insert": map[string]interface{}{"future-provider": map[string]interface{}{"version": 2, "reference": "opaque"}}},
		{"insert": "\nTexto final\n"},
	}
	raw, _ := json.Marshal(ops)
	if err := ValidateQuillDelta(string(raw)); err != nil {
		t.Fatal(err)
	}
	if !HasExtendedQuillBlocks(string(raw)) {
		t.Fatal("extended document must be protected from old editors")
	}
	if text := QuillPlainText(string(raw)); strings.Contains(text, "genially") || !strings.Contains(text, "Texto final") {
		t.Fatalf("unexpected plain text %q", text)
	}
	if err := ValidateQuillDelta(`[{"insert":{"video":"javascript:alert(1)"}}]`); err == nil {
		t.Fatal("invalid video accepted")
	}
	if err := ValidateQuillDelta(`[{"insert":{"video":"https://youtu.be/dQw4w9WgXcQ","onlife":"anything"}}]`); err == nil {
		t.Fatal("multi-key embed accepted")
	}
}

func TestIntegrationPreviewIsNotDuplicated(t *testing.T) {
	u := "https://padlet.com/embed/abcdefghijklmnop"
	var ops []map[string]interface{}
	_ = json.Unmarshal([]byte(integrationDelta("padlet", u)), &ops)
	ops = append(ops, map[string]interface{}{"insert": u + "\n"})
	raw, _ := json.Marshal(ops)
	if got := QuillPreviewURLs(string(raw), ""); len(got) != 0 {
		t.Fatalf("duplicate preview %v", got)
	}
}
