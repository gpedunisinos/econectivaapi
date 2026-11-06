package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestPlaceSearchGlobalCacheAndRate(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("countrycodes") != "" {
			t.Error("country restriction")
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing identification")
		}
		w.Write([]byte(`[{"display_name":"Lisboa, Portugal","lat":"38.7","lon":"-9.1"}]`))
	}))
	defer upstream.Close()
	s := NewPlaceSearch()
	s.Endpoint = upstream.URL
	if _, err := s.Search(context.Background(), "Lisboa Portugal"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Search(context.Background(), " lisboa   portugal "); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("cache miss")
	}
	if _, err := s.Search(context.Background(), "Paris France"); !errors.Is(err, ErrPlaceBusy) {
		t.Fatal("missing global rate limit", err)
	}
	if _, err := s.Search(context.Background(), "ab"); err == nil {
		t.Fatal("short query accepted")
	}
}
func TestPlaceSearchRejectsProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`not json`)) }))
	defer server.Close()
	s := NewPlaceSearch()
	s.Endpoint = server.URL
	if _, err := s.Search(context.Background(), "Tokyo Japan"); err == nil {
		t.Fatal("malformed response accepted")
	}
}
