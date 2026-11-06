package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/assert"
)

func TestFileFindAll(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	livingClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	file := models.File{
		Extension:    "jpg",
		Path:         "uploads/users/1/profile_picture.jpg",
		PublicURL:    "http://localhost:8080/storage/uploads/users/1/profile_picture.jpg",
		Name:         "profile_picture",
		LivingClueID: &livingClue.ID,
	}
	config.DB.Create(&file)

	request, _ := http.NewRequest("GET", "/api/files", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "OK response is expected")

	var files []models.File
	_ = json.Unmarshal(recorder.Body.Bytes(), &files)
	assert.NotNil(t, files, "response body must be a valid json")
	assert.Greater(t, len(files), 0)
}

func TestFileCreate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	livingClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)

	input := map[string]interface{}{
		"base_64":        "iVBORw0KGgoAAAANSUhEUgAAABkAAAASCAYAAACuLnWgAAAABHNCSVQICAgIfAhkiAAAABl0RVh0U29mdHdhcmUAZ25vbWUtc2NyZWVuc2hvdO8Dvz4AAAAmdEVYdENyZWF0aW9uIFRpbWUAc2V4IDE3IGphbiAyMDI1IDE0OjQ3OjI5crPYYQAAACNJREFUOI1j/P///38GGgMmWlswasmoJaOWjFoyasmoJdQCAMo0BCDSTR6+AAAAAElFTkSuQmCC",
		"extension":      "png",
		"path":           "uploads/users/1/",
		"name":           "profile_picture",
		"living_clue_id": livingClue.ID,
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", "/api/files", bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "OK response is expected")

	var file models.File
	_ = json.Unmarshal(recorder.Body.Bytes(), &file)
	assert.NotNil(t, file, "response body must be a valid json")
	assert.Contains(t, file.PublicURL, "/storage/")
	assert.True(t, strings.HasPrefix(file.Path, "files/"))
	_, statErr := os.Stat("storage/" + file.Path)
	assert.Nil(t, statErr)
}

func TestFileCreateWithUrl(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	input := map[string]interface{}{
		"base_64":        nil,
		"extension":      "png",
		"path":           "uploads/users/1/",
		"name":           "profile_picture",
		"public_url":     "https://example.com/image.jpg",
		"living_clue_id": nil,
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", "/api/files", bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var file models.File
	_ = json.Unmarshal(recorder.Body.Bytes(), &file)
	assert.NotNil(t, file, "Body: "+recorder.Body.String())
	assert.Equal(t, "https://example.com/image.jpg", file.PublicURL, "Body: "+recorder.Body.String())
}

func TestFileDelete(t *testing.T) {
	t.Setenv("APP_URL", "http://localhost:8080")
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	assert.Nil(t, os.MkdirAll("storage/uploads/users/1", 0o755))
	assert.Nil(t, os.WriteFile("storage/uploads/users/1/profile_picture.jpg", []byte("x"), 0o644))
	file := models.File{
		Extension: "jpg",
		Path:      "uploads/users/1/profile_picture.jpg",
		PublicURL: "http://localhost:8080/storage/uploads/users/1/profile_picture.jpg",
		Name:      "profile_picture",
	}
	config.DB.Create(&file)

	request, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/files/%d", file.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "OK response is expected")
	_, statErr := os.Stat("storage/uploads/users/1/profile_picture.jpg")
	assert.True(t, os.IsNotExist(statErr))
}
