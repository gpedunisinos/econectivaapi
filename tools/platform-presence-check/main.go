// Real transport checks with disposable accounts supplied in a protected file.
// Tokens and conversation text are never emitted in the report.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/OdairPianta/unisinos_usp_api/services/presence"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Platform struct {
	ID     string   `json:"id"`
	Base   string   `json:"base"`
	Origin string   `json:"origin"`
	Tokens []string `json:"tokens"`
}
type connected struct {
	conn *websocket.Conn
	self string
}

func must(err error) {
	if err != nil {
		panic("Platform presence verification failed (credentials omitted)")
	}
}
func check(ok bool) {
	if !ok {
		panic("Platform isolation assertion failed")
	}
}
func request(ctx context.Context, p Platform, method, path, token string, body any, status int) []byte {
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	r, err := http.NewRequestWithContext(ctx, method, p.Base+"/api"+path, bytes.NewReader(data))
	must(err)
	r.Header.Set("Origin", p.Origin)
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(r)
	must(err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	must(err)
	check(resp.StatusCode == status)
	return b
}
func open(ctx context.Context, p Platform, token string) connected {
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(p.Base, "http")+"/api/presence", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{p.Origin}}})
	must(err)
	must(wsjson.Write(ctx, c, map[string]string{"type": "auth", "token": token}))
	for {
		var e presence.Event
		must(wsjson.Read(ctx, c, &e))
		if e.Type == "snapshot" && e.Self != "" {
			return connected{c, e.Self}
		}
	}
}
func main() {
	check(len(os.Args) == 2)
	raw, err := os.ReadFile(os.Args[1])
	must(err)
	var platforms []Platform
	must(json.Unmarshal(raw, &platforms))
	check(len(platforms) == 2)
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	connections := make([][]connected, 2)
	defer func() {
		for _, list := range connections {
			for _, c := range list {
				_ = wsjson.Write(context.Background(), c.conn, map[string]string{"type": "bye"})
				c.conn.CloseNow()
			}
		}
	}()
	for i, p := range platforms {
		for _, token := range p.Tokens {
			c := open(ctx, p, token)
			connections[i] = append(connections[i], c)
			must(wsjson.Write(ctx, c.conn, map[string]any{"type": "state", "destination": "earth", "visible": true, "interacted": true}))
		}
		for {
			var e presence.Event
			must(wsjson.Read(ctx, connections[i][0].conn, &e))
			found := false
			for _, person := range e.Participants {
				if person.ID == connections[i][1].self && person.Destination == "earth" {
					found = true
				}
			}
			if found {
				break
			}
		}
	}
	check(connections[0][0].self != connections[1][0].self)
	for i, p := range platforms {
		other := platforms[1-i]
		// A participant's opaque ID from the other process is never resolvable here.
		request(ctx, p, "POST", "/me/conversations", p.Tokens[0], map[string]string{"participant": connections[1-i][1].self}, 404)
		raw := request(ctx, p, "POST", "/me/conversations", p.Tokens[0], map[string]string{"participant": connections[i][1].self}, 200)
		var convo struct {
			ID string `json:"id"`
		}
		must(json.Unmarshal(raw, &convo))
		check(convo.ID != "")
		message := "Validação técnica isolada 🙂 " + uuid.NewString()
		request(ctx, p, "POST", "/me/conversations/"+convo.ID+"/messages", p.Tokens[0], map[string]string{"text": message, "client_id": uuid.NewString()}, 200)
		received := request(ctx, p, "GET", "/me/conversations/"+convo.ID+"/messages", p.Tokens[1], nil, 200)
		check(bytes.Contains(received, []byte(message)))
		request(ctx, other, "GET", "/me/conversations/"+convo.ID+"/messages", other.Tokens[0], nil, 404)
		wsURL := "ws" + strings.TrimPrefix(p.Base, "http") + "/api/presence"
		c, r, e := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{other.Origin}}})
		if c != nil {
			c.CloseNow()
		}
		check(e != nil && r != nil && r.StatusCode == 403)
		wrong, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{p.Origin}}})
		must(err)
		must(wsjson.Write(ctx, wrong, map[string]string{"type": "auth", "token": other.Tokens[0]}))
		var event presence.Event
		err = wsjson.Read(ctx, wrong, &event)
		check(err != nil && websocket.CloseStatus(err) == websocket.StatusPolicyViolation)
		wrong.CloseNow()
	}
	fmt.Println(`{"presence_same_platform":true,"presence_cross_platform_rejected":true,"websocket_wrong_origin_rejected":true,"chat_text_and_emoji":true,"chat_cross_platform_rejected":true}`)
}
