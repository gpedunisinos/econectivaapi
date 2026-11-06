package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchEnvironmentalContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/context", request.URL.Path)
		assert.Equal(t, "Bearer test-token", request.Header.Get("Authorization"))
		var payload services.EnvironmentalContextRequest
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
		assert.Equal(t, -29.9, payload.Latitude)
		assert.Equal(t, -51.1, payload.Longitude)
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"biome":"Mata Atlântica",
			"land_cover":"Área urbanizada",
			"vegetation_percent":18.2,
			"water_distance_m":240,
			"altitude_m":732,
			"slope_percent":8.4,
			"built_up_percent":61,
			"population_density_km2":4270,
			"sources":{"mapbiomas":"collection-4-2025"}
		}`))
	}))
	defer server.Close()

	previousURL := os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL")
	previousToken := os.Getenv("ENVIRONMENTAL_CONTEXT_API_TOKEN")
	t.Cleanup(func() {
		_ = os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", previousURL)
		_ = os.Setenv("ENVIRONMENTAL_CONTEXT_API_TOKEN", previousToken)
	})
	require.NoError(t, os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", server.URL))
	require.NoError(t, os.Setenv("ENVIRONMENTAL_CONTEXT_API_TOKEN", "test-token"))

	result, err := services.FetchEnvironmentalContext(-29.9, -51.1)
	require.NoError(t, err)
	assert.Equal(t, "Mata Atlântica", result.Biome)
	assert.Equal(t, "Área urbanizada", result.LandCover)
	assert.Equal(t, 18.2, *result.VegetationPercent)
	assert.Equal(t, "collection-4-2025", result.Sources["mapbiomas"])
}

func TestFetchEnvironmentalContextRequiresEndpoint(t *testing.T) {
	previousURL := os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL")
	t.Cleanup(func() { _ = os.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", previousURL) })
	require.NoError(t, os.Unsetenv("ENVIRONMENTAL_CONTEXT_API_URL"))

	_, err := services.FetchEnvironmentalContext(-29.9, -51.1)
	assert.ErrorContains(t, err, "não configurada")
}
