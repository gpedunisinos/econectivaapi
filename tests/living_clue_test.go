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

func TestLivingClueFindAll(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	model, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	request, _ := http.NewRequest("GET", "/api/living_clues?search="+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.LivingClue
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}

func TestLivingClueCreate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	category, err := factories.CreateCategory()
	assert.Nil(t, err)
	delta := `[{"insert":"Descrição formatada"},{"insert":"\n","attributes":{"header":1}}]`
	payload := map[string]interface{}{"name": fakers.Name(), "category_id": category.ID, "description": "Descrição formatada", "description_delta": delta, "latitude": fakers.Latitude(), "longitude": fakers.Longitude(), "image_url": fakers.URL(), "metadata": map[string]interface{}{"extra": fakers.Word()}, "is_active": true}
	body, _ := json.Marshal(payload)
	request, _ := http.NewRequest("POST", "/api/living_clues", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result models.LivingClue
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, payload["name"], result.Name)
	assert.Equal(t, delta, result.DescriptionDelta)
	assert.NotNil(t, result.EnvironmentalContext)
	assert.Equal(t, models.EnvironmentalContextStatusPending, result.EnvironmentalContext.Status)
}

func TestLivingClueFind(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	model, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	for i := 0; i < 12; i++ {
		err = config.DB.Create(&models.SensorReading{LivingClueID: model.ID, ContentType: "application/json", PayloadRaw: fmt.Sprintf(`{"index":%d}`, i)}).Error
		assert.Nil(t, err)
	}
	request, _ := http.NewRequest("GET", "/api/living_clues/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result models.LivingClue
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
	assert.Len(t, result.SensorReadings, 10)
}

func TestLivingClueFindByBounds(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	category, err := factories.CreateCategory()
	assert.Nil(t, err)
	inside := models.LivingClue{Name: fakers.Name(), CategoryID: category.ID, Description: fakers.FullAddress(), Latitude: testCoordinate(-30.0), Longitude: testCoordinate(-51.0), ImageURL: fakers.URL(), Metadata: models.JSONMap{"info": "inside"}, IsActive: true}
	err = config.DB.Create(&inside).Error
	assert.Nil(t, err)
	request, _ := http.NewRequest("GET", "/api/living_clues?north=-29.0&south=-31.0&east=-50.0&west=-52.0&limit=100", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.LivingClue
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result, "Returned body: "+recorder.Body.String())
}

func TestLivingClueFindByRadius(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	category, err := factories.CreateCategory()
	assert.Nil(t, err)
	inside := models.LivingClue{Name: fakers.Name(), CategoryID: category.ID, Description: fakers.FullAddress(), Latitude: testCoordinate(-30.0), Longitude: testCoordinate(-51.0), ImageURL: fakers.URL(), Metadata: models.JSONMap{"info": "inside radius"}, IsActive: true}
	outside := models.LivingClue{Name: fakers.Name(), CategoryID: category.ID, Description: fakers.FullAddress(), Latitude: testCoordinate(10.0), Longitude: testCoordinate(10.0), ImageURL: fakers.URL(), Metadata: models.JSONMap{"info": "outside radius"}, IsActive: true}
	err = config.DB.Create(&inside).Error
	assert.Nil(t, err)
	err = config.DB.Create(&outside).Error
	assert.Nil(t, err)
	request, _ := http.NewRequest("GET", "/api/living_clues?lat=-30&lng=-51&radius=5000&limit=5", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
}

func TestLivingClueUpdate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	model, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	model.Name = fakers.Name()
	lat := fakers.Latitude()
	model.Latitude = &lat
	lng := fakers.Longitude()
	model.Longitude = &lng
	model.Description = "Descrição atualizada"
	model.DescriptionDelta = `[{"insert":"Descrição atualizada\n","attributes":{"bold":true}}]`
	body, _ := json.Marshal(model)
	request, _ := http.NewRequest("PUT", "/api/living_clues/"+fmt.Sprint(model.ID), bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result models.LivingClue
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.Name, result.Name)
	assert.Equal(t, model.Description, result.Description)
	assert.Equal(t, model.DescriptionDelta, result.DescriptionDelta)
	assert.NotNil(t, result.EnvironmentalContext)
	assert.Equal(t, models.EnvironmentalContextStatusPending, result.EnvironmentalContext.Status)
}

func TestLivingClueRejectsInvalidDescriptionDelta(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	category, err := factories.CreateCategory()
	assert.Nil(t, err)
	payload := map[string]interface{}{
		"name":              fakers.Name(),
		"category_id":       category.ID,
		"description":       "Descrição",
		"description_delta": "invalid json",
		"latitude":          fakers.Latitude(),
		"longitude":         fakers.Longitude(),
	}
	body, _ := json.Marshal(payload)
	request, _ := http.NewRequest("POST", "/api/living_clues", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Returned body: "+recorder.Body.String())
}

func TestLivingClueDelete(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	model, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	request, _ := http.NewRequest("DELETE", "/api/living_clues/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result models.LivingClue
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestLivingClueSelect(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	model, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	request, _ := http.NewRequest("GET", "/api/living_clues_select?limit=5&search="+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.ItemSelect
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}

func testCoordinate(v float64) *float64 { return &v }
