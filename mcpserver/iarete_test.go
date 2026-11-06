package mcpserver

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fixtureBackend struct{ calls int }

func (b *fixtureBackend) Execute(_ context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
	b.calls++
	return map[string]interface{}{"name": name, "arguments": args, "records": []interface{}{map[string]interface{}{"id": 42, "name": "Pista sintética"}}}, nil
}

func TestRealMCPClientQueriesOnlyDeclaredValidatedTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	backend := &fixtureBackend{}
	query, err := ConnectBackend(ctx, backend)
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()
	result, err := query.Call(ctx, "pista_obter", json.RawMessage(`{"id":42}`))
	if err != nil {
		t.Fatal(err)
	}
	if result["name"] != "pista_obter" || backend.calls != 1 {
		t.Fatalf("not executed through MCP: %v", result)
	}
	for _, call := range []struct{ name, args string }{
		{"pista_excluir", `{"id":42}`},
		{"pista_obter", `{}`},
		{"pista_obter", `{"id":42,"user_id":1}`},
		{"pistas_pesquisar", `{"limit":20000}`},
		{"pistas_pesquisar", `{"offset":-1}`},
	} {
		if _, err := query.Call(ctx, call.name, json.RawMessage(call.args)); err == nil {
			t.Fatalf("invalid query accepted: %s %s", call.name, call.args)
		}
	}
	if backend.calls != 1 {
		t.Fatal("invalid arguments reached backend")
	}
	tools, err := query.(*clientSession).client.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 7 {
		t.Fatalf("tools=%d", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Fatalf("tool not read-only: %s", tool.Name)
		}
	}
}
