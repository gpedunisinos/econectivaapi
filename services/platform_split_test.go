package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/models"
)

func TestSharedPlacesAcrossIndependentAPIs(t *testing.T) {
	t.Setenv("PLACE_SEARCH_SHARED_DIR", t.TempDir())
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`[{"display_name":"Lisboa","lat":"38.7","lon":"-9.1"}]`))
	}))
	defer server.Close()
	a, b, c := NewPlaceSearch(), NewPlaceSearch(), NewPlaceSearch()
	a.Endpoint = server.URL
	b.Endpoint = server.URL
	c.Endpoint = server.URL
	for _, client := range []*PlaceSearch{a, b} {
		if _, err := client.Search(context.Background(), "Lisboa Portugal"); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("shared cache missed: %d", calls)
	}
	if _, err := c.Search(context.Background(), "Tokyo Japan"); !errors.Is(err, ErrPlaceBusy) {
		t.Fatalf("cross-process rate limit bypassed: %v", err)
	}
}
func TestMeetingRoomsArePlatformScoped(t *testing.T) {
	t.Setenv("JITSI_PUBLIC_URL", "https://meet.example.org")
	t.Setenv("JITSI_APP_SECRET", strings.Repeat("test-meet-", 5))
	t.Setenv("JITSI_APP_ID", "test")
	user := models.User{DefaultModel: models.DefaultModel{ID: 1}, Name: "Test"}
	t.Setenv("PLATFORM_ID", "onlife")
	a, err := IssueMeetingAccess(user, "reuniao", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PLATFORM_ID", "atopos")
	b, err := IssueMeetingAccess(user, "reuniao", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if a.Room == b.Room || b.Room != "atopos-reuniao" {
		t.Fatalf("rooms not isolated: %s %s", a.Room, b.Room)
	}
	again, err := IssueMeetingAccess(user, b.Room, time.Now())
	if err != nil || again.Room != b.Room {
		t.Fatal("room prefix is not idempotent")
	}
}
