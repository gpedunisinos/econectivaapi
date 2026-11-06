package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/http/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBlockMiddlewareAllowsShortPathsWithoutPanicking(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, path := range []string{"/", "/a", "/api/status"} {
		t.Run(path, func(t *testing.T) {
			router := gin.New()
			router.Use(middlewares.BlockMiddleware())
			router.GET(path, func(context *gin.Context) {
				context.Status(http.StatusNoContent)
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)
			router.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusNoContent, recorder.Code)
		})
	}
}

func TestBlockMiddlewareRejectsProtectedPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, path := range []string{"/.git", "/.git/config", "/.env", "/.env.production"} {
		t.Run(path, func(t *testing.T) {
			router := gin.New()
			router.Use(middlewares.BlockMiddleware())
			router.GET("/*path", func(context *gin.Context) {
				context.Status(http.StatusNoContent)
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)
			router.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
			assert.JSONEq(t, `{"message":"Você não tem permissão para acessar este recurso"}`, recorder.Body.String())
		})
	}
}
