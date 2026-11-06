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
	"github.com/stretchr/testify/assert"
)

func TestSensorReadingCreateWithJSONPayload(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	livingClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)

	payload := []byte(`{"temperature": 23.5, "humidity": 78}`)
	request, _ := http.NewRequest("POST", "/api/living_clues/"+fmt.Sprint(livingClue.ID)+"/sensor_readings", bytes.NewBuffer(payload))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusCreated, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.SensorReading
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, livingClue.ID, result.LivingClueID)
	assert.Equal(t, "application/json", result.ContentType)
	assert.JSONEq(t, string(payload), result.PayloadRaw)
}

func TestSensorReadingCreateWithCSVPayload(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	livingClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)

	payload := []byte("timestamp,temperature\n2026-04-21T13:00:00Z,24.7")
	request, _ := http.NewRequest("POST", "/api/living_clues/"+fmt.Sprint(livingClue.ID)+"/sensor_readings", bytes.NewBuffer(payload))
	request.Header.Set("Content-Type", "text/csv")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusCreated, recorder.Code, "Returned body: "+recorder.Body.String())

	var persisted models.SensorReading
	err = config.DB.Where("living_clue_id = ?", livingClue.ID).First(&persisted).Error
	assert.Nil(t, err)
	assert.Equal(t, "text/csv", persisted.ContentType)
	assert.Equal(t, string(payload), persisted.PayloadRaw)
}

func TestSensorReadingFindAll(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	livingClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	for i := 0; i < 2; i++ {
		err = config.DB.Create(&models.SensorReading{LivingClueID: livingClue.ID, ContentType: "application/json", PayloadRaw: fmt.Sprintf(`{"index":%d}`, i)}).Error
		assert.Nil(t, err)
	}

	request, _ := http.NewRequest("GET", "/api/sensor_readings?limit=10", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.SensorReading
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.GreaterOrEqual(t, len(result), 2)
}

func TestSensorReadingFindOne(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	livingClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	sensorReading := models.SensorReading{LivingClueID: livingClue.ID, ContentType: "text/plain", PayloadRaw: "abc"}
	err = config.DB.Create(&sensorReading).Error
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/sensor_readings/"+fmt.Sprint(sensorReading.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result models.SensorReading
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, sensorReading.ID, result.ID)
}

func TestSensorReadingDelete(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	livingClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	sensorReading := models.SensorReading{LivingClueID: livingClue.ID, ContentType: "text/plain", PayloadRaw: "abc"}
	err = config.DB.Create(&sensorReading).Error
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", "/api/sensor_readings/"+fmt.Sprint(sensorReading.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var deleted models.SensorReading
	err = config.DB.Unscoped().Where("id = ?", sensorReading.ID).First(&deleted).Error
	assert.Nil(t, err)
	assert.NotNil(t, deleted.DeletedAt)
}
