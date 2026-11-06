package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/services/presence"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/require"
)

func TestPresenceAuthenticatedTwoAccountsAndMultipleTabs(t *testing.T) {
	setupDatabase()
	t.Setenv("PRESENCE_ALLOWED_ORIGINS", "http://presence.test")
	router := routesSetup()
	server := httptest.NewServer(router)
	defer server.Close()
	u, a := initUser()
	v, b := initUser()
	require.NoError(t, config.DB.Model(&u).Update("name", "Bruna Silva").Error)
	require.NoError(t, config.DB.Model(&v).Update("name", "Bruna Moraes").Error)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	open := func(token string) (*websocket.Conn, presence.Event) {
		c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/presence", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{"http://presence.test"}}})
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = wsjson.Write(ctx, c, map[string]string{"type": "bye"})
			c.Close(websocket.StatusNormalClosure, "")
		})
		require.NoError(t, wsjson.Write(ctx, c, map[string]string{"type": "auth", "token": token}))
		for {
			var e presence.Event
			require.NoError(t, wsjson.Read(ctx, c, &e))
			if e.Type == "snapshot" {
				return c, e
			}
		}
	}
	one, initial := open(a)
	self := initial.Self
	require.NotEmpty(t, self)
	two, other := open(b)
	require.NotEqual(t, self, other.Self)
	_, tab := open(a)
	require.Equal(t, self, tab.Self)
	count := 0
	for _, p := range tab.Participants {
		if p.ID == self {
			count++
		}
	}
	require.Equal(t, 1, count)
	require.NoError(t, wsjson.Write(ctx, two, map[string]interface{}{"type": "state", "destination": "earth", "visible": true, "interacted": true, "user_id": u.ID, "name": "Forged"}))
	for {
		var e presence.Event
		require.NoError(t, wsjson.Read(ctx, one, &e))
		found := false
		for _, p := range e.Participants {
			if p.ID == other.Self && p.Destination == "earth" {
				require.Equal(t, "Bruna M.", p.Name)
				found = true
			}
		}
		if found {
			break
		}
	}
	require.NoError(t, wsjson.Write(ctx, two, map[string]interface{}{"type": "state", "destination": "research", "visible": true, "interacted": true}))
	for {
		var e presence.Event
		require.NoError(t, wsjson.Read(ctx, one, &e))
		found := false
		for _, p := range e.Participants {
			if p.ID == other.Self && p.Destination == "research" {
				found = true
			}
		}
		if found {
			break
		}
	}
	require.NoError(t, wsjson.Write(ctx, two, map[string]interface{}{"type": "bye"}))
	for {
		var e presence.Event
		require.NoError(t, wsjson.Read(ctx, one, &e))
		found := false
		for _, id := range e.Removed {
			if id == other.Self {
				found = true
			}
		}
		if found {
			break
		}
	}
}
func TestPresenceRejectsBadOriginQueryCredentialsAndUnauthenticatedFrames(t *testing.T) {
	setupDatabase()
	t.Setenv("PRESENCE_ALLOWED_ORIGINS", "http://presence.test")
	server := httptest.NewServer(routesSetup())
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/presence"
	for _, tc := range []struct {
		origin, path string
		status       int
	}{{"https://evil.test", endpoint, 403}, {"http://presence.test", endpoint + "?token=not-accepted", 400}} {
		c, response, err := websocket.Dial(ctx, tc.path, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{tc.origin}}})
		if c != nil {
			c.CloseNow()
		}
		require.Error(t, err)
		require.Equal(t, tc.status, response.StatusCode)
	}
	for _, auth := range []map[string]interface{}{{"type": "state", "user_id": 1, "name": "Forged"}, {"type": "auth", "token": "invalid"}} {
		c, _, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{"http://presence.test"}}})
		require.NoError(t, err)
		require.NoError(t, wsjson.Write(ctx, c, auth))
		var event presence.Event
		err = wsjson.Read(ctx, c, &event)
		require.Error(t, err)
		require.Equal(t, websocket.StatusPolicyViolation, websocket.CloseStatus(err))
		c.CloseNow()
	}
}
