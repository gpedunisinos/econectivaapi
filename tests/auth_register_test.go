package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/stretchr/testify/assert"
)

func TestUserRegister(t *testing.T) {
	setupDatabase()
	router := routesSetup()

	email := fmt.Sprintf("register-%d@example.test", time.Now().UnixNano())
	body, _ := json.Marshal(map[string]string{"name": "Cadastro sintético", "email": email, "password": "123456"})
	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code, "OK response is expected")

	var result map[string]interface{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result["token"], "token must be not empty")
	assert.Equal(t, false, result["is_google_auth"])

	var user models.User
	err := config.DB.Where("email = ?", email).First(&user).Error
	assert.Nil(t, err)
	assert.Equal(t, enums.UserProfileEnumUser, user.Profile)
}

func TestUserRegisterBlockedWhenEmailBelongsToGoogleAccount(t *testing.T) {
	setupDatabase()
	router := routesSetup()

	email := fmt.Sprintf("google-%d@example.test", time.Now().UnixNano())
	err := config.DB.Create(&models.User{
		Name:         "Google User",
		Email:        email,
		Password:     "google-auth-only",
		Profile:      enums.UserProfileEnumUser,
		IsGoogleAuth: true,
	}).Error
	assert.Nil(t, err)

	body, _ := json.Marshal(map[string]string{"name": "Cadastro sintético", "email": email, "password": "123456"})
	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Bad request response is expected")
	assert.Contains(t, recorder.Body.String(), "Entre com Google")
}

func TestUserLoginBlockedWhenEmailBelongsToGoogleAccount(t *testing.T) {
	setupDatabase()
	router := routesSetup()

	email := fmt.Sprintf("google-%d@example.test", time.Now().UnixNano())
	err := config.DB.Create(&models.User{
		Name:         "Google User",
		Email:        email,
		Password:     "google-auth-only",
		Profile:      enums.UserProfileEnumUser,
		IsGoogleAuth: true,
	}).Error
	assert.Nil(t, err)

	body, _ := json.Marshal(map[string]string{"email": email, "password": "123456"})
	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Bad request response is expected")
	assert.Contains(t, recorder.Body.String(), "Entre com Google")
}
