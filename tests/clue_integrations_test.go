package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/require"
)

func TestClueIntegrationsPersistReopenEditRemoveAndProtectLegacyEditors(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, auth := initUser()
	_, explorer := initUser()
	request := func(auth, method, path string, input interface{}, status int) []byte {
		t.Helper()
		data, _ := json.Marshal(input)
		req := httptest.NewRequest(method, "/api"+path, bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer "+auth)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, status, response.Code, response.Body.String())
		return response.Body.Bytes()
	}
	request(auth, "PUT", "/me/preferences", map[string]string{"usage_mode": "inventor"}, 200)
	request(explorer, "PUT", "/me/preferences", map[string]string{"usage_mode": "explorer"}, 200)
	category, err := factories.CreateCategory()
	require.NoError(t, err)
	ops := []map[string]interface{}{{"insert": "Descrição de uma pista sintética\n"}}
	for _, block := range []map[string]interface{}{
		{"video": "https://youtu.be/dQw4w9WgXcQ"},
		{"onlife": `{"v":1,"provider":"genially","url":"https://view.genially.com/60e5417ee87d780d572d8c48"}`},
		{"onlife": `{"v":1,"provider":"padlet","url":"https://padlet.com/embed/abcdefghijklmnop","title":"Mural"}`},
		{"onlife": `{"v":1,"provider":"framevr","url":"https://framevr.io/example-room"}`},
		{"onlife": `{"v":1,"provider":"overte","url":"hifi://world.example.org"}`},
		{"onlife": `{"v":1,"provider":"vircadia","url":"hifi://world.example.org"}`},
		{"onlife": `{"v":1,"provider":"agent","url":"/space/agents/503","title":"Agente da plataforma"}`},
		{"onlife": `{"v":1,"provider":"node","url":"/nodes/graph?focus=503","title":"Problemática vinculada"}`},
		{"future-provider": map[string]interface{}{"v": 2, "reference": "preservar"}},
	} {
		ops = append(ops, map[string]interface{}{"insert": block}, map[string]interface{}{"insert": "\n"})
	}
	delta, _ := json.Marshal(ops)
	body := map[string]interface{}{"name": "Pista sintética de integrações", "category_id": category.ID, "description_delta": string(delta), "description_format_version": 1}
	var clue models.LivingClue
	require.NoError(t, json.Unmarshal(request(auth, "POST", "/living_clues", body, 200), &clue))
	path := fmt.Sprintf("/living_clues/%d", clue.ID)
	require.NoError(t, json.Unmarshal(request(explorer, "GET", path, nil, 200), &clue))
	require.JSONEq(t, string(delta), clue.DescriptionDelta)
	require.Equal(t, "Descrição de uma pista sintética", clue.Description)
	// Old clients must not silently remove embeds they cannot understand.
	delete(body, "description_format_version")
	request(auth, "PUT", path, body, http.StatusConflict)
	body["description_format_version"] = 1
	request(explorer, "PUT", path, body, http.StatusForbidden)
	// Edit the mural, preserving every other block and its order.
	ops[5]["insert"] = map[string]interface{}{"onlife": `{"v":1,"provider":"padlet","url":"https://padlet.com/embed/zyxwvutsrqponmlk","title":"Mural alterado"}`}
	delta, _ = json.Marshal(ops)
	body["description_delta"] = string(delta)
	request(auth, "PUT", path, body, 200)
	require.NoError(t, json.Unmarshal(request(auth, "GET", path, nil, 200), &clue))
	require.JSONEq(t, string(delta), clue.DescriptionDelta)
	// Removing one integration does not remove the remaining references.
	ops = append(ops[:5], ops[7:]...)
	delta, _ = json.Marshal(ops)
	body["description_delta"] = string(delta)
	request(auth, "PUT", path, body, 200)
	require.NoError(t, json.Unmarshal(request(auth, "GET", path, nil, 200), &clue))
	require.JSONEq(t, string(delta), clue.DescriptionDelta)
	request("", "GET", path, nil, http.StatusUnauthorized)
}

func TestLinkedAgentLookupBeyondFirstPageAndDeletedAgent(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	owner, _ := initUser()
	_, auth := initUser()
	agent := models.ChatTemplate{Name: "Agente vinculado", SystemInstruction: "Agente sintético", UserID: owner.ID}
	require.NoError(t, config.DB.Create(&agent).Error)
	newer := make([]models.ChatTemplate, 205)
	for i := range newer {
		newer[i] = models.ChatTemplate{Name: fmt.Sprintf("Agente novo %d", i), SystemInstruction: "Agente sintético", UserID: owner.ID}
	}
	require.NoError(t, config.DB.Create(&newer).Error)
	get := func(path, token string, status int) []byte {
		req := httptest.NewRequest("GET", "/api"+path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, status, response.Code, response.Body.String())
		return response.Body.Bytes()
	}
	path := fmt.Sprintf("/chat_templates/%d", agent.ID)
	get(path, "", 401)
	var found models.ChatTemplate
	require.NoError(t, json.Unmarshal(get(path, auth, 200), &found))
	require.Equal(t, agent.ID, found.ID)
	var page []models.ChatTemplate
	require.NoError(t, json.Unmarshal(get("/chat_templates?limit=200", auth, 200), &page))
	for _, item := range page {
		require.NotEqual(t, agent.ID, item.ID)
	}
	get("/chat_templates/invalid", auth, 400)
	require.NoError(t, config.DB.Delete(&agent).Error)
	get(path, auth, 404)
}
