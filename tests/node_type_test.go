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

func TestNodeTypeFindAll(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNodeType()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/node_types?search="+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.NodeType
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}

func TestNodeTypeCreate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	payload := models.NodeType{
		Name:  fakers.UUID(),
		Color: fakers.HexColor(),
	}

	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/node_types", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.NodeType
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, payload.Name, result.Name)
	assert.Equal(t, payload.Color, result.Color)
}

func TestNodeTypeFind(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNodeType()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/node_types/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.NodeType
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestNodeTypeUpdate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNodeType()
	assert.Nil(t, err)

	model.Name = fakers.UUID()
	model.Color = fakers.HexColor()

	body, _ := json.Marshal(model)

	request, _ := http.NewRequest("PUT", "/api/node_types/"+fmt.Sprint(model.ID), bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.NodeType
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.Name, result.Name)
	assert.Equal(t, model.Color, result.Color)
}

func TestNodeTypeDelete(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateNodeType()
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", "/api/node_types/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.NodeType
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}
