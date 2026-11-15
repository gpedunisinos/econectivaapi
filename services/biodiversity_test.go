package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBiodiversityParameters(t *testing.T) {
	for _, raw := range []string{"latitude=NaN&longitude=0", "latitude=91&longitude=0", "latitude=0&longitude=0&radius_m=999", "latitude=0&longitude=Inf"} {
		p, _ := url.ParseQuery(raw)
		if _, err := BiodiversityParameters(p, true); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	p, _ := url.ParseQuery("latitude=0&longitude=0&radius_m=500&private=true&per_page=99999")
	got, err := BiodiversityParameters(p, true)
	if err != nil || len(got) != 3 || got.Get("radius_m") != "500" {
		t.Fatal(got, err)
	}
	p, _ = url.ParseQuery("north=80&south=-80&east=180&west=-180")
	if _, err := BiodiversityParameters(p, false); err == nil {
		t.Fatal("unbounded viewport accepted")
	}
}

func TestFetchBiodiversityProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer private-geo-token" {
			t.Error("missing server token")
		}
		if r.URL.Path != "/biodiversity/nearby" || r.URL.Query().Get("radius_m") != "1000" {
			t.Error(r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"source":"iNaturalist","observations":[]}`))
	}))
	defer server.Close()
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", server.URL)
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_TOKEN", "private-geo-token")
	p, _ := BiodiversityParameters(url.Values{"latitude": {"0"}, "longitude": {"0"}}, true)
	body, err := FetchBiodiversity(context.Background(), p, true)
	if err != nil || string(body) != `{"source":"iNaturalist","observations":[]}` {
		t.Fatal(string(body), err)
	}
}

func TestFetchBiodiversityOutage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", server.URL)
	if _, err := FetchBiodiversity(context.Background(), url.Values{}, false); err == nil {
		t.Fatal("outage reported as success")
	}
}
