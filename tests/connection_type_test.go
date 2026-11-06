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

func TestConnectionTypeFindAll(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateConnectionType()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/connection_types?search="+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.ConnectionType
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}

func TestConnectionTypeCreate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	payload := models.ConnectionType{
		Name:          fakers.UUID(),
		StrokeStyle:   "dashed",
		IsDirectional: fakers.Bool(),
	}

	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/connection_types", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.ConnectionType
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, payload.Name, result.Name)
}

func TestConnectionTypeFind(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateConnectionType()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/connection_types/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.ConnectionType
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestConnectionTypeUpdate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateConnectionType()
	assert.Nil(t, err)

	model.Name = fakers.UUID()
	model.StrokeStyle = "dotted"
	model.IsDirectional = !model.IsDirectional

	body, _ := json.Marshal(model)

	request, _ := http.NewRequest("PUT", "/api/connection_types/"+fmt.Sprint(model.ID), bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.ConnectionType
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.Name, result.Name)
	assert.Equal(t, model.IsDirectional, result.IsDirectional)
}

func TestConnectionTypeDelete(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateConnectionType()
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", "/api/connection_types/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.ConnectionType
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestConnectionTypeSelect(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateConnectionType()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/connection_types_select?limit=5&search="+model.Name, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.ItemSelect
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}
