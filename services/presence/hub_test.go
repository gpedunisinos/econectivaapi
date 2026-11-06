package presence

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func snapshot(h *Hub) []Person { h.mu.Lock(); defer h.mu.Unlock(); return h.people() }
func TestSessionsArbitrateByInteractionNotHeartbeatAndExpire(t *testing.T) {
	h := NewHub()
	now := time.Now()
	a := h.Join(1, "Bruna Silva", now)
	b := h.Join(1, "Bruna Silva", now)
	other := h.Join(2, "Caio Rocha", now)
	h.Update(a.ID, "earth", true, true, now)
	h.Update(b.ID, "research", true, true, now)
	h.Update(a.ID, "earth", true, false, now.Add(time.Second))
	people := snapshot(h)
	if len(people) != 2 {
		t.Fatal(people)
	}
	var id string
	for _, p := range people {
		if p.Name == "Bruna" {
			id = p.ID
			if p.Destination != "research" {
				t.Fatal(p)
			}
		}
	}
	if id == "" {
		t.Fatal("missing identity")
	}
	h.Update(a.ID, "earth", true, true, now.Add(2*time.Second))
	h.Remove(b.ID)
	for _, p := range snapshot(h) {
		if p.ID == id && p.Destination != "earth" {
			t.Fatal(p)
		}
	}
	h.Detach(a.ID)
	for _, p := range snapshot(h) {
		if p.ID == id && (p.Destination != "earth" || p.Activity != "Na plataforma") {
			t.Fatal(p)
		}
	}
	h.Update(other.ID, "agents", true, true, now.Add(80*time.Second))
	h.Sweep(now.Add(93 * time.Second))
	if p := snapshot(h); len(p) != 1 || p[0].Name != "Caio" {
		t.Fatal(p)
	}
	h.Remove(other.ID)
	if len(snapshot(h)) != 0 {
		t.Fatal("presence leaked")
	}
}
func TestNamesAreReducedAndInternalIDsNeverDistributed(t *testing.T) {
	h := NewHub()
	now := time.Now()
	a := h.Join(12345, "Bruna Silva", now)
	h.Join(54321, "Bruna Moraes", now)
	h.Join(98765, "Bruna Souza", now)
	h.Update(a.ID, "observatory", true, true, now)
	names := map[string]bool{}
	for _, p := range snapshot(h) {
		if names[p.Name] {
			t.Fatal("ambiguous", p)
		}
		names[p.Name] = true
		if strings.Contains(p.Name, "Silva") || strings.Contains(p.Name, "Moraes") || strings.Contains(p.Name, "Souza") {
			t.Fatal(p)
		}
	}
	raw, _ := json.Marshal(snapshot(h))
	for _, secret := range []string{"12345", "54321", "98765", "user_id", "email", "token"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("private data", secret)
		}
	}
	if h.Update(a.ID, "/living_clues/42", true, true, now) {
		t.Fatal("arbitrary route accepted")
	}
}
func TestVersionedEventsAndSlowReadersRecoverWithSnapshot(t *testing.T) {
	h := NewHub()
	a := h.Join(1, "A", time.Now())
	var first Event
	for len(a.Out) > 0 {
		first = <-a.Out
	}
	if first.Type != "snapshot" || first.Self == "" {
		t.Fatal(first)
	}
	for i := 0; i < 40; i++ {
		d := "earth"
		if i%2 == 0 {
			d = "research"
		}
		h.Update(a.ID, d, true, true, time.Now())
	}
	found := false
	var last uint64
	for len(a.Out) > 0 {
		e := <-a.Out
		if e.Type == "snapshot" {
			found = true
		}
		if e.Version < last {
			t.Fatal("out of order")
		}
		last = e.Version
	}
	if !found {
		t.Fatal("slow consumer did not recover")
	}
}
func TestThirtyParticipantsReuseStableIdentities(t *testing.T) {
	h := NewHub()
	now := time.Now()
	sessions := []*Session{}
	for i := 1; i <= 30; i++ {
		s := h.Join(uint(i), "Pessoa Teste", now)
		sessions = append(sessions, s)
		h.Update(s.ID, "earth", true, true, now)
	}
	before := snapshot(h)
	if len(before) != 30 {
		t.Fatal(len(before))
	}
	for _, s := range sessions {
		h.Update(s.ID, "earth", true, false, now.Add(time.Second))
	}
	after := snapshot(h)
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	if string(a) != string(b) {
		t.Fatal("heartbeats changed fleet")
	}
}

func TestConnectionLimitAndExplicitLeavePreservesOtherTabs(t *testing.T) {
	h := NewHub()
	sessions := []*Session{}
	for i := 0; i < 12; i++ {
		s := h.Join(1, "Pessoa", time.Now())
		if s == nil {
			t.Fatal("early limit")
		}
		sessions = append(sessions, s)
	}
	if h.Join(1, "Pessoa", time.Now()) != nil {
		t.Fatal("session cap bypassed")
	}
	h.Remove(sessions[0].ID)
	if len(snapshot(h)) != 1 {
		t.Fatal("closing tab removed user")
	}
	if h.Join(1, "Pessoa", time.Now()) == nil {
		t.Fatal("slot not released")
	}
}

func TestPrivateChatNotificationsReachOnlyTheTwoAccounts(t *testing.T) {
	h := NewHub()
	a := h.Join(1, "Ana", time.Now())
	b := h.Join(2, "Bia", time.Now())
	tab := h.Join(1, "Ana", time.Now())
	outsider := h.Join(3, "Caio", time.Now())
	for _, s := range []*Session{a, b, tab, outsider} {
		for len(s.Out) > 0 {
			<-s.Out
		}
	}
	if user, ok := h.ResolveParticipant(h.users[2].ID); !ok || user != 2 {
		t.Fatal("opaque participant cannot be resolved")
	}
	h.NotifyConversation(1, 2, "private-conversation")
	for _, s := range []*Session{a, b, tab} {
		if len(s.Out) != 1 {
			t.Fatal("missing targeted notification")
		}
		event := <-s.Out
		if event.Type != "chat" || event.ConversationID != "private-conversation" {
			t.Fatal(event)
		}
	}
	if len(outsider.Out) != 0 {
		t.Fatal("private conversation leaked to another participant")
	}
}
