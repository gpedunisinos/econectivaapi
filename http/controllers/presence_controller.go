package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services/presence"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

var presenceHub = presence.NewHub()
var presenceStart sync.Once
var presenceSlots = make(chan struct{}, 512)

type presenceFrame struct {
	Type        string `json:"type"`
	Token       string `json:"token,omitempty"`
	Destination string `json:"destination,omitempty"`
	Visible     bool   `json:"visible"`
	Interacted  bool   `json:"interacted"`
}

func presenceOriginAllowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	configured := os.Getenv("PRESENCE_ALLOWED_ORIGINS")
	for _, allowed := range strings.Split(configured, ",") {
		if raw == strings.TrimSpace(allowed) && allowed != "" {
			return true
		}
	}
	if configured != "" {
		return false
	}
	return raw == "https://educacaoonlife.gpedu.com.br" || raw == "https://web.uniusp.gpedu.com.br" || raw == "https://plataforma.atopos.spotec.app"
}
func presenceIdentity(raw string) (uint, string, time.Time, error) {
	parsed, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, websocket.CloseError{Code: websocket.StatusPolicyViolation}
		}
		return []byte(os.Getenv("API_SECRET")), nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return 0, "", time.Time{}, websocket.CloseError{Code: websocket.StatusPolicyViolation}
	}
	c, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", time.Time{}, websocket.CloseError{Code: websocket.StatusPolicyViolation}
	}
	id, ok := c["user_id"].(float64)
	exp, hasExp := c["exp"].(float64)
	if !ok || id < 1 || id != float64(uint(id)) || !hasExp || exp <= float64(time.Now().Unix()) {
		return 0, "", time.Time{}, websocket.CloseError{Code: websocket.StatusPolicyViolation}
	}
	u, err := repository.FindUserByID(strconv.FormatUint(uint64(id), 10))
	if err != nil {
		return 0, "", time.Time{}, err
	}
	return u.ID, u.Name, time.Unix(int64(exp), 0), nil
}

// OpenPresence godoc
// @Summary Presença temporária das naves (WebSocket)
// @Description Upgrade com origem autorizada e sem token na URL. Primeiro frame JSON auth com token da sessão, em até 5 segundos. Após autenticar: snapshot, updates versionados e alive. Frames state/heartbeat aceitam apenas seção genérica; bye encerra a sessão. Sem acesso antes da autenticação. Uma nave por usuário, expiração em 90 segundos, uso prioritário web.
// @Tags presence
// @Success 101 {string} string "WebSocket; autenticação obrigatória no primeiro frame"
// @Failure 400 {object} models.APIMessage
// @Failure 403 {object} models.APIMessage
// @Failure 503 {object} models.APIMessage
// @Router /presence [get]
func OpenPresence(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if c.Request.URL.RawQuery != "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if !presenceOriginAllowed(c.GetHeader("Origin")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	select {
	case presenceSlots <- struct{}{}:
		defer func() { <-presenceSlots }()
	default:
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: []string{c.GetHeader("Origin")}})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(4096)
	authCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	var frame presenceFrame
	err = wsjson.Read(authCtx, conn, &frame)
	cancel()
	if err != nil || frame.Type != "auth" {
		conn.Close(websocket.StatusPolicyViolation, "Autenticação necessária")
		return
	}
	user, name, expires, err := presenceIdentity(frame.Token)
	frame.Token = ""
	if err != nil {
		conn.Close(websocket.StatusPolicyViolation, "Sessão inválida")
		return
	}
	presenceStart.Do(func() {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for now := range ticker.C {
				presenceHub.Sweep(now)
			}
		}()
	})
	s := presenceHub.Join(user, name, time.Now())
	if s == nil {
		conn.Close(websocket.StatusPolicyViolation, "Limite de sessões de presença")
		return
	}
	out := s.Out
	explicit := false
	defer func() {
		if explicit {
			presenceHub.Remove(s.ID)
		} else {
			presenceHub.Detach(s.ID)
		}
	}()
	ctx, stop := context.WithDeadline(context.Background(), expires)
	defer stop()
	go func() {
		defer stop()
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-out:
				writeCtx, done := context.WithTimeout(ctx, 5*time.Second)
				err := wsjson.Write(writeCtx, conn, event)
				done()
				if err != nil {
					return
				}
			}
		}
	}()
	lastSeen := time.Now()
	window := time.Now()
	frames := 0
	for {
		readCtx, done := context.WithTimeout(ctx, presence.TTL)
		kind, raw, readErr := conn.Read(readCtx)
		done()
		if readErr != nil {
			if time.Now().After(expires) || time.Since(lastSeen) >= presence.TTL {
				explicit = true
			}
			return
		}
		if time.Since(window) > 10*time.Second {
			window = time.Now()
			frames = 0
		}
		frames++
		frame = presenceFrame{}
		if frames > 30 || kind != websocket.MessageText || json.Unmarshal(raw, &frame) != nil {
			explicit = true
			conn.Close(websocket.StatusPolicyViolation, "Mensagem inválida")
			return
		}
		if frame.Type == "bye" {
			explicit = true
			conn.Close(websocket.StatusNormalClosure, "Até breve")
			return
		}
		if (frame.Type != "state" && frame.Type != "heartbeat") || !presenceHub.Update(s.ID, frame.Destination, frame.Visible, frame.Type == "state" && frame.Interacted, time.Now()) {
			explicit = true
			conn.Close(websocket.StatusPolicyViolation, "Seção inválida")
			return
		}
		lastSeen = time.Now()
		if frame.Type == "heartbeat" {
			writeCtx, finish := context.WithTimeout(ctx, 5*time.Second)
			err = wsjson.Write(writeCtx, conn, map[string]string{"type": "alive"})
			finish()
			if err != nil {
				return
			}
		}
	}
}
