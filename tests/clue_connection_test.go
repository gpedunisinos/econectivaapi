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
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
	"github.com/stretchr/testify/assert"
)

func TestClueConnectionFindAll(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateClueConnection()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/clue_connections?search="+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.ClueConnection
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}

func TestFindLivingClueConnections(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	outgoing, err := factories.CreateClueConnection()
	assert.Nil(t, err)
	incoming, err := factories.CreateClueConnection()
	assert.Nil(t, err)
	incoming.TargetClueID = outgoing.SourceClueID
	assert.Nil(t, config.DB.Save(incoming).Error)

	request, _ := http.NewRequest(
		"GET",
		"/api/living_clues/"+fmt.Sprint(outgoing.SourceClueID)+"/connections",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.ClueConnection
	assert.Nil(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	assert.Len(t, result, 2)
	for _, connection := range result {
		assert.NotNil(t, connection.SourceClue)
		assert.NotNil(t, connection.TargetClue)
		assert.NotNil(t, connection.ConnectionType)
	}
}

func TestFindLivingClueConnectionsNotFound(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	request, _ := http.NewRequest("GET", "/api/living_clues/999999/connections", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestClueConnectionCreate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	sourceClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	targetClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	connectionType, err := factories.CreateConnectionType()
	assert.Nil(t, err)

	payload := map[string]interface{}{
		"source_clue_id":     sourceClue.ID,
		"target_clue_id":     targetClue.ID,
		"connection_type_id": connectionType.ID,
		"strength":           2,
		"evidence":           fakers.FullAddress(),
	}

	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/clue_connections", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.ClueConnection
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, payload["source_clue_id"], result.SourceClueID)
}

func TestClueConnectionFind(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateClueConnection()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/clue_connections/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.ClueConnection
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestClueConnectionUpdate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateClueConnection()
	assert.Nil(t, err)

	model.Strength = 5
	model.Evidence = fakers.FullAddress()

	body, _ := json.Marshal(model)

	request, _ := http.NewRequest("PUT", "/api/clue_connections/"+fmt.Sprint(model.ID), bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.ClueConnection
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.Strength, result.Strength)
	assert.Equal(t, model.Evidence, result.Evidence)
}

func TestClueConnectionDelete(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateClueConnection()
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", "/api/clue_connections/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.ClueConnection
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestClueConnectionAvoidDuplicates(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateClueConnection()
	assert.Nil(t, err)

	payload := map[string]interface{}{
		"source_clue_id":     model.SourceClueID,
		"target_clue_id":     model.TargetClueID,
		"connection_type_id": model.ConnectionTypeID,
		"strength":           1,
	}
	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/clue_connections", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Returned body: "+recorder.Body.String())
}
