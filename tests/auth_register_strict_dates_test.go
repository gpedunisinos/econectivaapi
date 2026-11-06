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
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/stretchr/testify/require"
)

func TestRegisterUnverifiedEmailUnderStrictDates(t *testing.T) {
	setupDatabase()
	original := config.DB
	tx := original.Begin()
	require.NoError(t, tx.Error)
	config.DB = tx
	var previousMode string
	require.NoError(t, tx.Raw("SELECT @@SESSION.sql_mode").Scan(&previousMode).Error)
	t.Cleanup(func() {
		tx.Exec("SET SESSION sql_mode = ?", previousMode)
		tx.Rollback()
		config.DB = original
	})
	require.NoError(t, tx.Exec("SET SESSION sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE'").Error)
	router := routesSetup()
	email := fmt.Sprintf("strict-register-%d@example.test", time.Now().UnixNano())
	body, _ := json.Marshal(map[string]string{"name": "Cadastro sintético", "email": email, "password": "Synthetic-Test-428!"})
	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	require.NotEmpty(t, result["token"])
	require.Nil(t, result["email_verified_at"])
	user, err := repository.FindUserByEmail(email)
	require.NoError(t, err)
	require.Nil(t, user.EmailVerifiedAt)
	var nullDates int64
	require.NoError(t, tx.Model(&models.User{}).Where("id = ? AND email_verified_at IS NULL", user.ID).Count(&nullDates).Error)
	require.EqualValues(t, 1, nullDates)
	// Later saves must keep an unverified timestamp NULL as well.
	require.NoError(t, repository.UpdateFcmToken(user, "synthetic-device-token"))
	verifiedAt := time.Now().UTC().Truncate(time.Millisecond)
	google := models.User{Name: "Conta verificada sintética", Email: "verified-" + email, Password: "not-used", Profile: enums.UserProfileEnumUser, IsGoogleAuth: true, EmailVerifiedAt: &verifiedAt}
	require.NoError(t, repository.CreateUser(&google))
	saved, err := repository.FindUserByEmail(google.Email)
	require.NoError(t, err)
	require.NotNil(t, saved.EmailVerifiedAt)
	require.WithinDuration(t, verifiedAt, *saved.EmailVerifiedAt, time.Second)
}
