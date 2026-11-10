package cdig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaginationRejectsPartialErrorsAndRepeatedIDs(t *testing.T) {
	for _, mode := range []string{"complete", "partial", "duplicate", "errors", "unauthorized"} {
		t.Run(mode, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if mode == "unauthorized" {
					w.WriteHeader(401)
					fmt.Fprint(w, "do not expose synthetic-secret")
					return
				}
				if mode == "errors" {
					fmt.Fprint(w, `{"errors":[{"message":"do not expose synthetic-secret"}]}`)
					return
				}
				var input struct{ Query string }
				_ = json.NewDecoder(r.Body).Decode(&input)
				number := 1
				id := "1"
				more := true
				if strings.Contains(input.Query, "page:2") {
					number = 2
					id = "2"
					more = false
					if mode == "duplicate" {
						id = "1"
					}
				}
				rows := []map[string]any{{"id": id}}
				if mode == "partial" && number == 2 {
					rows = nil
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"resources": map[string]any{"data": rows, "paginatorInfo": map[string]any{"total": 2, "currentPage": number, "hasMorePages": more}}}})
			}))
			defer server.Close()
			client := NewClient(Config{BaseURL: server.URL})
			rows, err := fetchPages[Resource](context.Background(), client, "synthetic", "resources", "id", "")
			if mode == "complete" {
				if err != nil || len(rows) != 2 || requests != 2 {
					t.Fatalf("pagination: %v", err)
				}
			} else {
				if err == nil || strings.Contains(err.Error(), "synthetic-secret") {
					t.Fatal("invalid page accepted or remote error exposed")
				}
			}
		})
	}
}
func TestDisabledDoesNotCallSource(t *testing.T) {
	client := NewClient(Config{})
	if _, err := client.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	sync := Sync{Config: Config{}}
	report, err := sync.Run(context.Background(), false)
	if err != nil || report.Enabled {
		t.Fatal("disabled integration must require no dependencies")
	}
}
func TestTemporaryTokenAlwaysRevokedOnFetchFailure(t *testing.T) {
	revoked := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/integration/token" && r.Method == "POST" {
			fmt.Fprint(w, `{"token":"synthetic-temporary","scope":"content:read"}`)
			return
		}
		if r.Method == "DELETE" {
			revoked = r.Header.Get("Authorization") == "Bearer synthetic-temporary"
			fmt.Fprint(w, `{}`)
			return
		}
		w.WriteHeader(503)
	}))
	defer server.Close()
	client := NewClient(Config{Enabled: true, BaseURL: server.URL, Email: "synthetic", Password: "synthetic"})
	if _, err := client.Fetch(context.Background()); err == nil || !revoked {
		t.Fatal("failed fetch must revoke the temporary token")
	}
}
