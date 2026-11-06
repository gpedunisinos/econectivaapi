package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

func TestMeetingAccessAuthenticatedBothModes(t *testing.T) {
	setupDatabase()
	t.Setenv("JITSI_APP_SECRET", strings.Repeat("meeting-test-only-", 4))
	t.Setenv("JITSI_APP_ID", "onlife-test")
	t.Setenv("JITSI_PUBLIC_URL", "https://meet.example.org")
	t.Setenv("JITSI_XMPP_DOMAIN", "meet.jitsi")
	router := routesSetup()
	user, auth := initUserWithProfile(enums.UserProfileEnumUser)
	request := func(auth, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/me/meetings/access", bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
		if auth != "" {
			r.Header.Set("Authorization", "Bearer "+auth)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	require.Equal(t, http.StatusUnauthorized, request("", `{"room":"sala"}`).Code)
	for _, mode := range []string{"explorer", "inventor"} {
		require.NoError(t, config.DB.Model(&user).Update("usage_mode", mode).Error)
		w := request(auth, `{"room":"Sala Aurora","user_id":999,"moderator":false,"host":"evil.test"}`)
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		var data struct {
			Room    string `json:"room"`
			URL     string `json:"public_url"`
			JWT     string `json:"jwt"`
			Expires int64  `json:"expires_at"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &data))
		require.Equal(t, "sala-aurora", data.Room)
		require.Equal(t, "https://meet.example.org/sala-aurora", data.URL)
		require.NotContains(t, data.URL, "jwt")
		parsed, err := jwt.Parse(data.JWT, func(token *jwt.Token) (interface{}, error) {
			require.Equal(t, jwt.SigningMethodHS256, token.Method)
			return []byte(strings.Repeat("meeting-test-only-", 4)), nil
		})
		require.NoError(t, err)
		require.True(t, parsed.Valid)
		c := parsed.Claims.(jwt.MapClaims)
		require.Equal(t, "onlife-test", c["iss"])
		require.Equal(t, "onlife-test", c["aud"])
		require.Equal(t, "meet.jitsi", c["sub"])
		require.Equal(t, "sala-aurora", c["room"])
		require.InDelta(t, time.Now().Add(10*time.Minute).Unix(), data.Expires, 3)
		identity := c["context"].(map[string]interface{})["user"].(map[string]interface{})
		require.Equal(t, user.Name, identity["name"])
		require.Equal(t, true, identity["moderator"])
		require.NotEqual(t, "999", identity["id"])
		require.NotContains(t, w.Body.String(), user.Email)
		_, err = jwt.Parse(data.JWT, func(*jwt.Token) (interface{}, error) { return []byte("wrong-secret"), nil })
		require.Error(t, err)
	}
	for _, room := range []string{"*", "https://evil.test/room", "a/b", "", "static", strings.Repeat("a", 121)} {
		body, _ := json.Marshal(map[string]string{"room": room})
		require.Equal(t, 400, request(auth, string(body)).Code)
	}
	require.Equal(t, 400, request(auth, strings.Repeat("x", 5000)).Code)
	t.Setenv("JITSI_APP_SECRET", "")
	w := request(auth, `{"room":"sala"}`)
	require.Equal(t, 503, w.Code)
	require.NotContains(t, w.Body.String(), "jwt")
}
