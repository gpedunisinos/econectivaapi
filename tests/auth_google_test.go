package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/stretchr/testify/assert"
)

func TestUserLoginGoogleWithExistingUser(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, _ := initUser()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"aud":"google-web-client-id","email":"` + user.Email + `","name":"Google User"}`))
	}))
	defer server.Close()

	t.Setenv("GOOGLE_WEB_CLIENT_ID", "google-web-client-id")
	t.Setenv("GOOGLE_TOKENINFO_URL", server.URL)

	body := []byte(`{"id_token": "valid-google-id-token"}`)
	req, _ := http.NewRequest("POST", "/api/login/google", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code, "OK response is expected")

	var result map[string]string
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result["token"], "token must be not empty")
}

func TestUserLoginGoogleCreateUser(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	email := "google.created@example.com"

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"aud":"google-web-client-id","email":"` + email + `","name":"Created By Google"}`))
	}))
	defer server.Close()

	t.Setenv("GOOGLE_WEB_CLIENT_ID", "google-web-client-id")
	t.Setenv("GOOGLE_TOKENINFO_URL", server.URL)

	body := []byte(`{"id_token": "valid-google-id-token"}`)
	req, _ := http.NewRequest("POST", "/api/login/google", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code, "OK response is expected")

	createdUser, err := repository.FindUserByEmail(email)
	assert.Nil(t, err)
	assert.Equal(t, enums.UserProfileEnumUser, createdUser.Profile)

	var result map[string]string
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.NotEmpty(t, result["token"], "token must be not empty")
}

func TestUserLoginGoogleWithAccessToken(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	email := "google.access@example.com"

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"aud":"google-web-client-id","email":"` + email + `","name":"Access Token User"}`))
	}))
	defer server.Close()

	t.Setenv("GOOGLE_WEB_CLIENT_ID", "google-web-client-id")
	t.Setenv("GOOGLE_TOKENINFO_URL", server.URL)

	body := []byte(`{"access_token": "valid-google-access-token"}`)
	req, _ := http.NewRequest("POST", "/api/login/google", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code, "OK response is expected")

	createdUser, err := repository.FindUserByEmail(email)
	assert.Nil(t, err)
	assert.Equal(t, enums.UserProfileEnumUser, createdUser.Profile)
}

func TestUserLoginGoogleWithoutTokens(t *testing.T) {
	setupDatabase()
	router := routesSetup()

	body := []byte(`{}`)
	req, _ := http.NewRequest("POST", "/api/login/google", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Bad request response is expected")
}

func TestUserLoginGoogleWithInvalidAudience(t *testing.T) {
	setupDatabase()
	router := routesSetup()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"aud":"another-client-id","email":"test@example.com","name":"Wrong Audience"}`))
	}))
	defer server.Close()

	t.Setenv("GOOGLE_WEB_CLIENT_ID", "google-web-client-id")
	t.Setenv("GOOGLE_TOKENINFO_URL", server.URL)

	body := []byte(`{"id_token": "invalid-audience-token"}`)
	req, _ := http.NewRequest("POST", "/api/login/google", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code, "Unauthorized response is expected")
}
