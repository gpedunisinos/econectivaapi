package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/require"
)

func TestInventiveClassificationsJSONPersistence(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, auth := initUser()
	_, explorer := initUser()
	request := func(token, method, path string, body interface{}, status int) []byte {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(method, "/api"+path, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, status, res.Code, res.Body.String())
		return res.Body.Bytes()
	}
	request(auth, "PUT", "/me/preferences", map[string]string{"usage_mode": "inventor"}, 200)
	request(explorer, "PUT", "/me/preferences", map[string]string{"usage_mode": "explorer"}, 200)
	category, err := factories.CreateCategory()
	require.NoError(t, err)
	delta := `[{"insert":"Vivência sintética\n"},{"insert":{"onlife":"{\"v\":1,\"provider\":\"agent\",\"url\":\"/space/agents/503\"}"}},{"insert":"\n"}]`
	for _, located := range []bool{false, true} {
		t.Run(fmt.Sprint("located=", located), func(t *testing.T) {
			metadata := map[string]interface{}{"cidade": "Cidade sintética", "future": map[string]interface{}{"items": []interface{}{true, float64(3)}}, "praticas_inventivas": []interface{}{"niig", "saigon", "future_code"}, "metodologias_inventivas": []interface{}{"pag", "biblioteca_viva"}}
			body := map[string]interface{}{"name": "Classificação sintética", "category_id": category.ID, "metadata": metadata, "description_delta": delta, "description_format_version": 1}
			if located {
				body["latitude"] = -29.718
				body["longitude"] = -51.143
			}
			var clue models.LivingClue
			require.NoError(t, json.Unmarshal(request(auth, "POST", "/living_clues", body, 200), &clue))
			path := fmt.Sprintf("/living_clues/%d", clue.ID)
			read := func() {
				require.NoError(t, json.Unmarshal(request(explorer, "GET", path, nil, 200), &clue))
				require.Equal(t, models.JSONMap(metadata), clue.Metadata)
				require.JSONEq(t, delta, clue.DescriptionDelta)
				require.Equal(t, located, clue.Latitude != nil && clue.Longitude != nil)
			}
			read()
			request(explorer, "PUT", path, body, 403)
			// Updating only classifications keeps all other payload and metadata fields.
			metadata["praticas_inventivas"] = []interface{}{}
			request(auth, "PUT", path, body, 200)
			read()
			// Old, unknown and nullable values remain compatible; no silent inference.
			for _, legacy := range []interface{}{"NIIG antigo", map[string]interface{}{"legacy": true}, nil, []interface{}{"future_code"}} {
				metadata["metodologias_inventivas"] = legacy
				request(auth, "PUT", path, body, 200)
				read()
			}
		})
	}
}
