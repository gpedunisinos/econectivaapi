package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/routes"
	"github.com/OdairPianta/unisinos_usp_api/services/token"
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/stretchr/testify/assert"
)

func setupDatabase() (*gorm.DB, func()) {
	config.InitDatabase()
	tx := config.DB.Begin()

	cleanup := func() {
		tx.Rollback()
	}

	return tx, cleanup
}

func routesSetup() *gin.Engine {
	ginMode := os.Getenv("GIN_MODE")
	if ginMode != "" {
		gin.SetMode(ginMode)
	}
	r := gin.Default()
	routes.InitRoutes(r)
	return r
}

func initUser() (user models.User, stringToken string) {
	return initUserWithProfile(enums.UserProfileEnumAdministrator)
}

func initUserWithProfile(profile enums.UserProfileEnum) (user models.User, stringToken string) {
	verifiedAt := time.Now()
	user = models.User{
		Name:               fakers.Name(),
		Email:              fakers.Email(),
		Password:           fakers.Word(),
		EmailVerifiedAt:    &verifiedAt,
		RememberToken:      fakers.UUID(),
		FCMToken:           fakers.UUID(),
		Token:              fakers.UUID(),
		ResetPasswordToken: fakers.UUID(),
		Profile:            profile,
	}
	err := config.DB.Create(&user).Error
	if err != nil {
		fmt.Println(err)
		return
	}

	token, errToken := token.GenerateToken(user.ID)
	if errToken != nil {
		fmt.Println(err)
		return
	}
	return user, token
}

func TestRootPathReturnsNotFoundWithoutPanicking(t *testing.T) {
	router := routesSetup()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		router.ServeHTTP(recorder, request)
	})
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestFindGitHeadFilesReturnAccessDeniedResponse(t *testing.T) {
	setupDatabase()
	router := routesSetup()

	request, _ := http.NewRequest("GET", "/.git/HEAD", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code, "Unauthorized response is expected")

	var resultModel map[string]string
	_ = json.Unmarshal(recorder.Body.Bytes(), &resultModel)
	assert.NotNil(t, resultModel, "response body must be a valid json")
}

func TestFindGitConfigFilesReturnAccessDeniedResponse(t *testing.T) {
	setupDatabase()
	router := routesSetup()

	request, _ := http.NewRequest("GET", "/.git/config", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code, "Unauthorized response is expected")

	var resultModel map[string]string
	_ = json.Unmarshal(recorder.Body.Bytes(), &resultModel)
	assert.NotNil(t, resultModel, "response body must be a valid json")
}

func TestFindEnvFilesReturnAccessDeniedResponse(t *testing.T) {
	setupDatabase()
	router := routesSetup()

	request, _ := http.NewRequest("GET", "/.env", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code, "Unauthorized response is expected")

	var resultModel map[string]string
	_ = json.Unmarshal(recorder.Body.Bytes(), &resultModel)
	assert.NotNil(t, resultModel, "response body must be a valid json")
}
