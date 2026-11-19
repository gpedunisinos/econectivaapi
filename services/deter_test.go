package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDeterProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/deter/nearby" || r.Header.Get("Authorization") != "Bearer private" {
			t.Error("incorrect proxy request")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"covered":true,"total":2}`))
	}))
	defer server.Close()
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", server.URL)
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_TOKEN", "private")
	p, err := BiodiversityParameters(url.Values{"latitude": {"-3"}, "longitude": {"-55"}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = FetchDeter(context.Background(), p); err != nil {
		t.Fatal(err)
	}
}

func TestDeterRejectsUpstreamFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("not json")) }))
	defer server.Close()
	t.Setenv("ENVIRONMENTAL_CONTEXT_API_URL", server.URL)
	if _, err := FetchDeter(context.Background(), url.Values{}); err == nil {
		t.Fatal("invalid upstream accepted")
	}
}
