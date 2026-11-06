package services

import (
	"errors"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/dgrijalva/jwt-go"
)

var ErrMeetingUnavailable = errors.New("meeting authentication unavailable")
var ErrMeetingRoom = errors.New("invalid meeting room")
var meetingRoomPattern = regexp.MustCompile(`^[\pL\pN_-]+$`)

// MeetingAccess is a short-lived room credential, never a platform session token.
type MeetingAccess struct {
	Room      string `json:"room"`
	PublicURL string `json:"public_url"`
	JWT       string `json:"jwt"`
	ExpiresAt int64  `json:"expires_at"`
}

// IssueMeetingAccess authorizes every authenticated account regardless of usage mode.
// No caller-supplied identity, role, hostname or wildcard is accepted.
func IssueMeetingAccess(user models.User, room string, now time.Time) (MeetingAccess, error) {
	room = strings.ToLower(strings.Join(strings.Fields(room), "-"))
	if utf8.RuneCountInString(room) > 120 || !meetingRoomPattern.MatchString(room) {
		return MeetingAccess{}, ErrMeetingRoom
	}
	for _, reserved := range []string{"http-bind", "xmpp-websocket", "colibri-ws", "static", "libs", "images", "css", "login", "logout", "api"} {
		if room == reserved {
			return MeetingAccess{}, ErrMeetingRoom
		}
	}
	if platform := PlatformID(); platform != "" {
		// Idempotent when a participant pastes an already namespaced invite.
		if !strings.HasPrefix(room, platform+"-") {
			room = platform + "-" + room
		}
		if utf8.RuneCountInString(room) > 120 {
			return MeetingAccess{}, ErrMeetingRoom
		}
	}
	base, err := url.Parse(os.Getenv("JITSI_PUBLIC_URL"))
	secret, appID := os.Getenv("JITSI_APP_SECRET"), os.Getenv("JITSI_APP_ID")
	if user.ID == 0 || err != nil || base.Scheme != "https" || base.Hostname() == "" || base.User != nil || (base.Port() != "" && base.Port() != "443") || (base.Path != "" && base.Path != "/") || base.RawQuery != "" || base.Fragment != "" || len(secret) < 32 || secret == os.Getenv("API_SECRET") || appID == "" {
		return MeetingAccess{}, ErrMeetingUnavailable
	}
	domain := os.Getenv("JITSI_XMPP_DOMAIN")
	if domain == "" {
		domain = "meet.jitsi"
	}
	expires := now.Add(10 * time.Minute).Unix()
	claims := jwt.MapClaims{
		"iss": appID, "aud": appID, "sub": domain, "room": room,
		"iat": now.Unix(), "nbf": now.Add(-5 * time.Second).Unix(), "exp": expires,
		"context": map[string]interface{}{"user": map[string]interface{}{"id": strconv.FormatUint(uint64(user.ID), 10), "name": user.Name, "moderator": true}},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return MeetingAccess{}, ErrMeetingUnavailable
	}
	base.Path = "/" + room
	return MeetingAccess{Room: room, PublicURL: base.String(), JWT: signed, ExpiresAt: expires}, nil
}
