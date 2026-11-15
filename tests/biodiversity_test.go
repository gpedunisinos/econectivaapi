package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/stretchr/testify/require"
)

func TestFaunaAuthenticatedRoutes(t *testing.T) {
	_, cleanup := setupDatabase()
	defer cleanup()
	t.Setenv("TOKEN_HOUR_LIFESPAN", "24")
	_, token := initUserWithProfile(enums.UserProfileEnumUser)
	require.NotEmpty(t, token)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer geo-test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"source":"iNaturalist","observations":[],"limit":12}`))
	}))
	defer upstream.Close()
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", upstream.URL)
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_TOKEN", "geo-test-token")
	router := routesSetup()
	for _, route := range []string{"/api/biodiversity/nearby?latitude=0&longitude=0&radius_m=500", "/api/biodiversity/observations?north=1&south=0&east=1&west=0"} {
		unauth := httptest.NewRecorder()
		router.ServeHTTP(unauth, httptest.NewRequest("GET", route, nil))
		require.Equal(t, 401, unauth.Code)
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", route, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, req)
		require.Equal(t, 200, recorder.Code, recorder.Body.String())
		require.NotContains(t, recorder.Body.String(), "geo-test-token")
	}
}

func TestFaunaLiveGeoContract(t *testing.T) {
	endpoint := os.Getenv("FAUNA_LIVE_GEO_URL")
	if endpoint == "" {
		t.Skip("opt-in local Geo API integration")
	}
	_, cleanup := setupDatabase()
	defer cleanup()
	t.Setenv("TOKEN_HOUR_LIFESPAN", "24")
	_, token := initUserWithProfile(enums.UserProfileEnumUser)
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", endpoint)
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_TOKEN", "fauna-local-test-token")
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/biodiversity/nearby?latitude=-29.794&longitude=-51.151&radius_m=1000", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	routesSetup().ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code, recorder.Body.String())
	var result struct {
		Source       string `json:"source"`
		Observations []struct {
			ID      int    `json:"id"`
			License string `json:"photo_license"`
			Credit  string `json:"photo_attribution"`
		} `json:"observations"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	require.Equal(t, "iNaturalist", result.Source)
	require.NotEmpty(t, result.Observations)
	require.LessOrEqual(t, len(result.Observations), 12)
	for _, item := range result.Observations {
		require.NotEmpty(t, item.Credit)
		require.Contains(t, []string{"cc0", "cc-by", "cc-by-sa"}, item.License)
	}
	t.Logf("Go → Geo API → iNaturalist: %d species with credited licensed photos", len(result.Observations))
}
