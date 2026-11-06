package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/mcpserver"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/require"
)

type mcpSourceTransport func(*http.Request) (*http.Response, error)

func (f mcpSourceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestIareteMCPAuthenticatedQueriesAndPinnedSource(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()
	mode := "explorer"
	require.NoError(t, config.DB.Model(&user).Update("usage_mode", mode).Error)
	category, err := factories.CreateCategory()
	require.NoError(t, err)
	search := fmt.Sprintf("onlife-fixture-%d", time.Now().UnixNano())
	a := models.LivingClue{Name: search + " sem coordenadas", CategoryID: category.ID, Description: "Há 42 árvores nesta amostra sintética. https://source.example.org/fixture", IsActive: true, Metadata: models.JSONMap{"observacao": "sintética", "api_key": "synthetic-private-token"}}
	require.NoError(t, config.DB.Create(&a).Error)
	latitude, longitude := -29.0, -51.0
	b := models.LivingClue{Name: search + " localizada", CategoryID: category.ID, Description: "Outra pista sintética", Latitude: &latitude, Longitude: &longitude, IsActive: true}
	require.NoError(t, config.DB.Create(&b).Error)
	rpc := func(auth, method string, params interface{}, status int) map[string]interface{} {
		t.Helper()
		data, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		req := httptest.NewRequest("POST", "/api/mcp/iarete", bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer "+auth)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, status, response.Code, response.Body.String())
		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		return result
	}
	rpc("", "tools/list", map[string]interface{}{}, http.StatusUnauthorized)
	listed := rpc(token, "tools/list", map[string]interface{}{}, 200)
	require.Len(t, listed["result"].(map[string]interface{})["tools"], 7)
	backend := mcpserver.NewDatabaseBackend(user, 0)
	query, err := mcpserver.ConnectBackend(context.Background(), backend)
	require.NoError(t, err)
	defer query.Close()
	args, _ := json.Marshal(map[string]interface{}{"text": search, "limit": 10})
	results, err := query.Call(context.Background(), "pistas_pesquisar", args)
	require.NoError(t, err)
	require.Len(t, results["records"], 2, "non-geographic search must include both clues")
	args, _ = json.Marshal(map[string]interface{}{"text": search, "north": -28, "south": -30, "east": -50, "west": -52})
	results, err = query.Call(context.Background(), "pistas_pesquisar", args)
	require.NoError(t, err)
	require.Len(t, results["records"], 1)
	args, _ = json.Marshal(map[string]interface{}{"id": a.ID})
	details, err := query.Call(context.Background(), "pista_obter", args)
	require.NoError(t, err)
	require.Equal(t, false, details["has_location"])
	require.Nil(t, details["environment"])
	raw, _ := json.Marshal(details)
	require.NotContains(t, string(raw), "synthetic-private-token")
	require.Contains(t, string(raw), "42 árvores")
	backend.Sources = services.NewSourceFetcherWithTransport(mcpSourceTransport(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/plain"}}, Body: io.NopCloser(strings.NewReader("A fonte sintética informa que a amostra tem 42 árvores e foi registrada para teste.")), Request: req}, nil
	}))
	sources := services.ClueSources(&a)
	require.Len(t, sources, 1)
	args, _ = json.Marshal(map[string]interface{}{"id": a.ID, "source_id": sources[0].ID})
	result, err := query.Call(context.Background(), "pista_fonte_consultar", args)
	require.NoError(t, err)
	raw, _ = json.Marshal(result)
	require.Contains(t, string(raw), "42 árvores")
	require.Contains(t, string(raw), `"status":"ok"`)
	args, _ = json.Marshal(map[string]interface{}{"id": a.ID, "source_id": "https://unrelated.example.org"})
	_, err = query.Call(context.Background(), "pista_fonte_consultar", args)
	require.Error(t, err)
	pinned := mcpserver.NewDatabaseBackend(user, a.ID)
	_, err = pinned.Execute(context.Background(), "pista_obter", map[string]interface{}{"id": b.ID})
	require.Error(t, err, "clue chat may not swap its pinned identity")
	response := rpc(token, "tools/call", map[string]interface{}{"name": "pista_excluir", "arguments": map[string]interface{}{"id": a.ID}}, 200)
	if result, ok := response["result"].(map[string]interface{}); ok {
		require.Equal(t, true, result["isError"])
	} else {
		require.NotNil(t, response["error"])
	}
	var current models.LivingClue
	require.NoError(t, config.DB.First(&current, a.ID).Error)
	require.Equal(t, a.Name, current.Name)
}

func TestIareteMCPMeetingSourceDoesNotReadOrMonitorTheMeeting(t *testing.T) {
	setupDatabase()
	t.Setenv("JITSI_PUBLIC_URL", "https://meet.spotec.app")
	user, _ := initUser()
	category, err := factories.CreateCategory()
	require.NoError(t, err)
	clue := models.LivingClue{Name: "Encontro sintético", CategoryID: category.ID, Description: "Convite https://meet.spotec.app/sala-sintetica", IsActive: true}
	require.NoError(t, config.DB.Create(&clue).Error)
	backend := mcpserver.NewDatabaseBackend(user, clue.ID)
	backend.Sources = services.NewSourceFetcherWithTransport(mcpSourceTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("meeting source must never be fetched or monitored")
		return nil, nil
	}))
	session, err := mcpserver.ConnectBackend(context.Background(), backend)
	require.NoError(t, err)
	defer session.Close()
	sources := services.ClueSources(&clue)
	require.Len(t, sources, 1)
	args, _ := json.Marshal(map[string]interface{}{"id": clue.ID, "source_id": sources[0].ID})
	result, err := session.Call(context.Background(), "pista_fonte_consultar", args)
	require.NoError(t, err)
	require.Equal(t, "interactive_only", result["status"])
	require.Contains(t, result["limitation"], "não acessou nem acompanhou")
	require.Nil(t, result["source"])
}
