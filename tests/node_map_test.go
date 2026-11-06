package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/assert"
)

func TestNodesMap(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	_, err := factories.CreateNodeLink()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/nodes/map", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result map[string]json.RawMessage
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Contains(t, result, "nodes")
	assert.Contains(t, result, "links")

	var nodes []map[string]interface{}
	var links []map[string]interface{}
	_ = json.Unmarshal(result["nodes"], &nodes)
	_ = json.Unmarshal(result["links"], &links)
	assert.Greater(t, len(nodes), 0, "Returned body: "+recorder.Body.String())
	assert.Greater(t, len(links), 0, "Returned body: "+recorder.Body.String())
}
