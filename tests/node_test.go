package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
	"github.com/stretchr/testify/assert"
)

func TestNodeFindAll(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNode()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/nodes?search="+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.Node
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}

func TestNodeCreate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	nodeType, err := factories.CreateNodeType()
	assert.Nil(t, err)

	x := 123.45
	y := 678.90
	payload := map[string]interface{}{
		"name":         fakers.Name(),
		"color":        fakers.HexColor(),
		"x":            x,
		"y":            y,
		"node_type_id": nodeType.ID,
	}

	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/nodes", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.Node
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, payload["name"], result.Name)
	assert.Equal(t, payload["node_type_id"], result.NodeTypeID)
	assert.NotNil(t, result.PositionX)
	assert.NotNil(t, result.PositionY)
	assert.InDelta(t, payload["x"], *result.PositionX, 0.000001)
	assert.InDelta(t, payload["y"], *result.PositionY, 0.000001)
	assert.NotNil(t, result.NodeType)
}

func TestNodeFind(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNode()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/nodes/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.Node
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestNodeUpdate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNode()
	assert.Nil(t, err)

	newNodeType, err := factories.CreateNodeType()
	assert.Nil(t, err)

	x := 11.22
	y := 33.44
	payload := map[string]interface{}{
		"name":         fakers.Name(),
		"color":        fakers.HexColor(),
		"x":            x,
		"y":            y,
		"node_type_id": newNodeType.ID,
	}

	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("PUT", "/api/nodes/"+fmt.Sprint(model.ID), bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.Node
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, payload["name"], result.Name)
	assert.Equal(t, payload["node_type_id"], result.NodeTypeID)
	assert.NotNil(t, result.PositionX)
	assert.NotNil(t, result.PositionY)
	assert.InDelta(t, payload["x"], *result.PositionX, 0.000001)
	assert.InDelta(t, payload["y"], *result.PositionY, 0.000001)
}

func TestNodeDelete(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNode()
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", "/api/nodes/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.Node
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}
