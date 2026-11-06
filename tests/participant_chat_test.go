package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/OdairPianta/unisinos_usp_api/services/presence"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParticipantChatPrivacyPersistenceEmojisPaginationAndRetries(t *testing.T) {
	setupDatabase()
	t.Setenv("PRESENCE_ALLOWED_ORIGINS", "http://chat.test")
	server := httptest.NewServer(routesSetup())
	defer server.Close()
	a, ta := initUser()
	_, tb := initUser()
	_, tc := initUser()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	open := func(token string) (*websocket.Conn, string) {
		c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/presence", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{"http://chat.test"}}})
		require.NoError(t, err)
		deferCleanup := c
		t.Cleanup(func() { deferCleanup.CloseNow() })
		require.NoError(t, wsjson.Write(ctx, c, map[string]string{"type": "auth", "token": token}))
		for {
			var e presence.Event
			require.NoError(t, wsjson.Read(ctx, c, &e))
			if e.Type == "snapshot" {
				return c, e.Self
			}
		}
	}
	_, selfA := open(ta)
	receiver, selfB := open(tb)
	require.NotEqual(t, selfA, selfB)
	request := func(token, method, path string, input interface{}, status int) map[string]interface{} {
		t.Helper()
		raw, _ := json.Marshal(input)
		req, err := http.NewRequest(method, server.URL+"/api"+path, bytes.NewReader(raw))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer response.Body.Close()
		var data map[string]interface{}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&data))
		require.Equal(t, status, response.StatusCode, data)
		return data
	}
	request(ta, "PUT", "/me/preferences", map[string]string{"usage_mode": "explorer"}, 200)
	request(ta, "POST", "/me/conversations", map[string]string{"participant": selfA}, 404)
	request(ta, "POST", "/me/conversations", map[string]string{"participant": "1"}, 404)
	convo := request(ta, "POST", "/me/conversations", map[string]string{"participant": selfB}, 200)
	id := convo["id"].(string)
	path := "/me/conversations/" + id
	repeated := request(tb, "POST", "/me/conversations", map[string]string{"participant": selfA}, 200)
	require.Equal(t, id, repeated["id"])
	request(tc, "GET", path+"/messages", nil, 404)
	request(tc, "POST", path+"/messages", map[string]string{"text": "Não permitido", "client_id": uuid.NewString()}, 404)
	request("", "GET", path+"/messages", nil, 401)
	clientID := uuid.NewString()
	text := "Olá, vamos explorar juntos? 😊🚀🌱❤️"
	sent := request(ta, "POST", path+"/messages", map[string]interface{}{"text": text, "client_id": clientID, "sender_id": 999}, 200)
	require.Equal(t, true, sent["own"])
	retry := request(ta, "POST", path+"/messages", map[string]string{"text": text, "client_id": clientID}, 200)
	require.Equal(t, sent["id"], retry["id"])
	request(ta, "POST", path+"/messages", map[string]string{"text": "Outro texto", "client_id": clientID}, 409)
	for _, invalid := range []string{"   ", strings.Repeat("a", 2001), "bad\x00text"} {
		request(ta, "POST", path+"/messages", map[string]string{"text": invalid, "client_id": uuid.NewString()}, 400)
	}
	// Only metadata is delivered over presence; the private text stays in the HTTP API.
	for {
		var event map[string]interface{}
		require.NoError(t, wsjson.Read(ctx, receiver, &event))
		if event["type"] == "chat" {
			require.Equal(t, id, event["conversation_id"])
			raw, _ := json.Marshal(event)
			require.NotContains(t, string(raw), "Olá")
			require.NotContains(t, string(raw), "sender_id")
			break
		}
	}
	page := request(tb, "GET", path+"/messages", nil, 200)
	rows := page["messages"].([]interface{})
	require.Len(t, rows, 1)
	message := rows[0].(map[string]interface{})
	require.Equal(t, text, message["text"])
	require.Equal(t, false, message["own"])
	require.NotContains(t, message, "sender_id")
	require.NotContains(t, message, "email")
	inbox := request(tb, "GET", "/me/conversations", nil, 200)
	require.Equal(t, float64(1), inbox["unread"])
	request(tb, "POST", path+"/read", map[string]interface{}{"through": sent["id"]}, 200)
	inbox = request(tb, "GET", "/me/conversations", nil, 200)
	require.Equal(t, float64(0), inbox["unread"])
	outsider := request(tc, "GET", "/me/conversations", nil, 200)
	require.Empty(t, outsider["conversations"])
	// Use synthetic rows to test history without bypassing the send-rate contract.
	for i := 0; i < 35; i++ {
		require.NoError(t, config.DB.Create(&models.ParticipantMessage{ConversationID: id, SenderID: a.ID, ClientID: uuid.NewString(), Text: fmt.Sprintf("Histórico %02d ✨", i)}).Error)
	}
	page = request(tb, "GET", path+"/messages", nil, 200)
	rows = page["messages"].([]interface{})
	require.Len(t, rows, 30)
	require.Equal(t, true, page["has_more"])
	oldest := uint64(rows[0].(map[string]interface{})["id"].(float64))
	older := request(tb, "GET", fmt.Sprintf("%s/messages?before=%d", path, oldest), nil, 200)
	require.Len(t, older["messages"], 6)
}

func TestParticipantChatSendRateDoesNotErasePersistedMessages(t *testing.T) {
	setupDatabase()
	user, auth := initUser()
	peer, _ := initUser()
	conversation, err := services.OpenParticipantConversation(user.ID, peer.ID)
	require.NoError(t, err)
	router := routesSetup()
	for i := 0; i < 21; i++ {
		raw, _ := json.Marshal(map[string]string{"text": "Mensagem sintética 🚀", "client_id": uuid.NewString()})
		req := httptest.NewRequest("POST", "/api/me/conversations/"+conversation.ID+"/messages", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+auth)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		expected := 200
		if i == 20 {
			expected = 429
		}
		require.Equal(t, expected, response.Code, response.Body.String())
	}
	var count int64
	require.NoError(t, config.DB.Model(&models.ParticipantMessage{}).Where("conversation_id = ?", conversation.ID).Count(&count).Error)
	require.Equal(t, int64(20), count)
}

func TestParticipantConversationReportAndBlock(t *testing.T) {
	setupDatabase()
	a, tokenA := initUser()
	b, tokenB := initUser()
	_, outsiderToken := initUser()
	conversation, err := services.OpenParticipantConversation(a.ID, b.ID)
	require.NoError(t, err)
	router := routesSetup()
	path := "/api/me/conversations/" + conversation.ID
	request := func(token, method, suffix string, body interface{}, status int) {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(method, path+suffix, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, status, response.Code, response.Body.String())
	}
	request(outsiderToken, "POST", "/report", map[string]string{"description": "irrelevant"}, 404)
	request(tokenA, "POST", "/report", map[string]string{"description": "  Assédio  "}, 201)
	var report models.ParticipantConversationReport
	require.NoError(t, config.DB.Where("conversation_id = ?", conversation.ID).First(&report).Error)
	require.Equal(t, a.ID, report.UserID)
	require.Equal(t, "Assédio", *report.Description)
	request(tokenA, "POST", "/block", nil, 200)
	request(tokenA, "POST", "/block", nil, 200)
	request(tokenB, "GET", "/messages", nil, 403)
	request(tokenB, "POST", "/messages", map[string]string{"text": "Olá", "client_id": uuid.NewString()}, 403)
	request(tokenA, "POST", "/messages", map[string]string{"text": "Olá", "client_id": uuid.NewString()}, 403)
	request(tokenA, "DELETE", "/block", nil, 200)
	request(tokenB, "GET", "/messages", nil, 200)
	request(tokenB, "POST", "/messages", map[string]string{"text": "Olá", "client_id": uuid.NewString()}, 200)
}
