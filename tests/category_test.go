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

func TestCategoryFindAll(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateCategory()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/categories?search="+model.Name, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.Category
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}

func TestCategoryCreate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	payload := models.Category{
		Name:         fakers.Company(),
		Type:         fakers.Word(),
		ColorHex:     fakers.HexColor(),
		IconCode:     fakers.Word(),
		Description:  fakers.FullAddress(),
		MetadataTags: fakers.Word(),
	}

	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/categories", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.Category
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, payload.Name, result.Name)
	assert.Equal(t, payload.Type, result.Type)
}

func TestCategoryFind(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateCategory()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/categories/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.Category
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestCategoryCreateRequiresType(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	payload := map[string]interface{}{
		"name":          fakers.Company(),
		"color_hex":     fakers.HexColor(),
		"icon_code":     fakers.Word(),
		"description":   fakers.FullAddress(),
		"metadata_tags": fakers.Word(),
	}

	body, _ := json.Marshal(payload)

	request, _ := http.NewRequest("POST", "/api/categories", bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Returned body: "+recorder.Body.String())
	assert.Contains(t, recorder.Body.String(), "Type")
}

func TestCategoryUpdate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateCategory()
	assert.Nil(t, err)

	model.Name = fakers.Company()
	model.Type = fakers.Word()
	model.ColorHex = fakers.HexColor()
	model.MetadataTags = fakers.Word()

	body, _ := json.Marshal(model)

	request, _ := http.NewRequest("PUT", "/api/categories/"+fmt.Sprint(model.ID), bytes.NewBuffer(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.Category
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.Name, result.Name)
	assert.Equal(t, model.Type, result.Type)
}

func TestCategoryDelete(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateCategory()
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", "/api/categories/"+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())

	var result models.Category
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Equal(t, model.ID, result.ID)
}

func TestCategorySelect(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateCategory()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/categories_select?search="+model.Name, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.ItemSelect
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}
