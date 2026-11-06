package tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/OdairPianta/unisinos_usp_api/routes"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchEnvironmentalLayersRewritesTileURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "Bearer secret", request.Header.Get("Authorization"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[{"id":"biomes","name":"Biomas","description":"","data_type":"vector","tile_format":"png","tile_url":"/tiles/biomes/{z}/{x}/{y}.png","default_opacity":0.35,"min_zoom":3,"max_zoom":12,"available":true,"legend":[],"source":{}}]`))
	}))
	defer server.Close()
	previousURL := os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL")
	previousToken := os.Getenv("ENVIRONMENTAL_CONTEXT_API_TOKEN")
	t.Cleanup(func() {
		_ = os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", previousURL)
		_ = os.Setenv("ENVIRONMENTAL_CONTEXT_API_TOKEN", previousToken)
	})
	require.NoError(t, os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", server.URL))
	require.NoError(t, os.Setenv("ENVIRONMENTAL_CONTEXT_API_TOKEN", "secret"))

	layers, err := services.FetchEnvironmentalLayers()
	require.NoError(t, err)
	require.Len(t, layers, 1)
	assert.Equal(t, "/api/environmental_tiles/biomes/{z}/{x}/{y}.png", layers[0].TileURL)
}

func TestFetchEnvironmentalLayersPreservesJPEGFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[{"id":"satellite","name":"Imagem de satélite","description":"","data_type":"raster","tile_format":"jpg","tile_url":"/tiles/satellite/{z}/{x}/{y}.jpg","map_role":"basemap","attribution":"Imagem © Esri","default_opacity":1,"min_zoom":0,"max_zoom":19,"available":true,"legend":[],"source":{}}]`))
	}))
	defer server.Close()
	previousURL := os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL")
	t.Cleanup(func() { _ = os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", previousURL) })
	require.NoError(t, os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", server.URL))

	layers, err := services.FetchEnvironmentalLayers()
	require.NoError(t, err)
	require.Len(t, layers, 1)
	assert.Equal(t, "basemap", layers[0].MapRole)
	assert.Equal(t, "/api/environmental_tiles/satellite/{z}/{x}/{y}.jpg", layers[0].TileURL)
}

func TestFetchEnvironmentalTileForwardsJPEGPathAndToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/tiles/satellite/8/91/150.jpg", request.URL.Path)
		assert.Equal(t, "Bearer geo-secret", request.Header.Get("Authorization"))
		writer.Header().Set("Content-Type", "image/jpeg")
		_, _ = writer.Write([]byte("jpeg"))
	}))
	defer server.Close()
	previousURL := os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL")
	previousToken := os.Getenv("ENVIRONMENTAL_CONTEXT_API_TOKEN")
	t.Cleanup(func() {
		_ = os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", previousURL)
		_ = os.Setenv("ENVIRONMENTAL_CONTEXT_API_TOKEN", previousToken)
	})
	require.NoError(t, os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", server.URL))
	require.NoError(t, os.Setenv("ENVIRONMENTAL_CONTEXT_API_TOKEN", "geo-secret"))

	response, err := services.FetchEnvironmentalTile("satellite", 8, 91, 150, "jpg")
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, "image/jpeg", response.Header.Get("Content-Type"))
}

func TestEnvironmentalTileRouteParsesYCoordinateWithPNGExtension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/tiles/satellite/4/6/9.jpg", request.URL.Path)
		writer.Header().Set("Content-Type", "image/jpeg")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("tile"))
	}))
	defer server.Close()
	previousURL := os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL")
	t.Cleanup(func() { _ = os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", previousURL) })
	require.NoError(t, os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", server.URL))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api")
	routes.RegisterEnvironmentalLayerRoutes(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/environmental_tiles/satellite/4/6/9.png",
		nil,
	)

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "image/jpeg", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "tile", recorder.Body.String())
}

func TestFindEnvironmentalLayersReturnsEmptyListWhenServiceIsUnavailable(t *testing.T) {
	previousURL := os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL")
	t.Cleanup(func() { _ = os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", previousURL) })
	require.NoError(t, os.Unsetenv("ENVIRONMENTAL_CONTEXT_API_URL"))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/environmental_layers", controllers.FindEnvironmentalLayers)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/environmental_layers", nil)

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `[]`, recorder.Body.String())
}
