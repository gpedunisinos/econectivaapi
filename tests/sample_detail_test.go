package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/assert"
)

func TestSampleDetailSelect(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	model, err := factories.CreateSampleDetail()
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/sample_details_select?search="+fmt.Sprint(model.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Returned body: "+recorder.Body.String())
	var result []models.ItemSelect
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Greater(t, len(result), 0, "Returned body: "+recorder.Body.String())
}
