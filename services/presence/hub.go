// Package presence holds ephemeral, authenticated section presence, never coordinates or content.
package presence

import (
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const TTL = 90 * time.Second

type Person struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Destination string `json:"destination"`
	Activity    string `json:"activity"`
}
type Event struct {
	Type           string   `json:"type"`
	Version        uint64   `json:"version"`
	Self           string   `json:"self,omitempty"`
	Participants   []Person `json:"participants"`
	Removed        []string `json:"removed,omitempty"`
	ConversationID string   `json:"conversation_id,omitempty"`
}
type Session struct {
	ID          string
	User        uint
	FullName    string
	Destination string
	Visible     bool
	Seen        time.Time
	Interaction uint64
	Out         chan Event
}
type identity struct{ ID, Name, Last string }
type Hub struct {
	mu                sync.Mutex
	sessions          map[string]*Session
	users             map[uint]*identity
	current           map[string]Person
	version, sequence uint64
}

func NewHub() *Hub {
	return &Hub{sessions: map[string]*Session{}, users: map[uint]*identity{}, current: map[string]Person{}}
}
func ValidDestination(s string) bool {
	return s == "cockpit" || s == "earth" || s == "research" || s == "observatory" || s == "agents" || s == "platform"
}
func Activity(d string) string {
	return map[string]string{"cockpit": "No cockpit", "earth": "Explorando o mapa", "research": "Explorando pistas", "observatory": "Explorando problemáticas", "agents": "Explorando agentes", "platform": "Na plataforma"}[d]
}
func (h *Hub) Join(user uint, name string, now time.Time) *Session {
	h.mu.Lock()
	defer h.mu.Unlock()
	count := 0
	for _, existing := range h.sessions {
		if existing.User == user {
			count++
		}
	}
	if count >= 12 {
		return nil
	}
	if h.users[user] == nil {
		h.users[user] = &identity{ID: uuid.NewString(), Name: name, Last: "cockpit"}
	}
	h.users[user].Name = name
	s := &Session{ID: uuid.NewString(), User: user, FullName: name, Destination: "platform", Seen: now, Out: make(chan Event, 16)}
	h.sessions[s.ID] = s
	h.publish()
	s.Out <- Event{Type: "snapshot", Version: h.version, Self: h.users[user].ID, Participants: h.people()}
	return s
}

// Update trusts only allowed sections; names and identities never come from a frame.
// Heartbeats refresh liveness but cannot win arbitration over a more recent interaction.
func (h *Hub) Update(id, destination string, visible, interacted bool, now time.Time) bool {
	if !ValidDestination(destination) {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.sessions[id]
	if s == nil {
		return false
	}
	s.Seen = now
	s.Visible = visible
	s.Destination = destination
	if visible && interacted {
		h.sequence++
		s.Interaction = h.sequence
	}
	h.publish()
	return true
}
func (h *Hub) Detach(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s := h.sessions[id]; s != nil {
		s.Visible = false
		s.Out = nil
		h.publish()
	}
}
func (h *Hub) Remove(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, id)
	h.publish()
}
func (h *Hub) Sweep(now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, s := range h.sessions {
		if now.Sub(s.Seen) >= TTL {
			delete(h.sessions, id)
		}
	}
	h.publish()
}
func (h *Hub) Count() int { h.mu.Lock(); defer h.mu.Unlock(); return len(h.sessions) }
func (h *Hub) people() []Person {
	p := make([]Person, 0, len(h.current))
	for _, v := range h.current {
		p = append(p, v)
	}
	sort.Slice(p, func(i, j int) bool { return p[i].ID < p[j].ID })
	return p
}
func firstName(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return "Explorador"
	}
	return string([]rune(f[0])[:min(40, len([]rune(f[0])))])
}
func (h *Hub) publish() {
	grouped := map[uint][]*Session{}
	for _, s := range h.sessions {
		grouped[s.User] = append(grouped[s.User], s)
	}
	for user := range h.users {
		if len(grouped[user]) == 0 {
			delete(h.users, user)
		}
	}
	names := map[string]int{}
	for _, i := range h.users {
		names[strings.ToLower(firstName(i.Name))]++
	}
	labels := map[uint]string{}
	duplicates := map[string][]uint{}
	for user, i := range h.users {
		n := firstName(i.Name)
		if names[strings.ToLower(n)] > 1 {
			f := strings.Fields(i.Name)
			if len(f) > 1 {
				n += " " + string([]rune(f[len(f)-1])[0]) + "."
			}
		}
		labels[user] = n
		duplicates[strings.ToLower(n)] = append(duplicates[strings.ToLower(n)], user)
	}
	for _, ids := range duplicates {
		if len(ids) > 1 {
			sort.Slice(ids, func(i, j int) bool { return h.users[ids[i]].ID < h.users[ids[j]].ID })
			for n, user := range ids {
				labels[user] += " · " + presentationNumber(n+1)
			}
		}
	}
	next := map[string]Person{}
	for user, sessions := range grouped {
		i := h.users[user]
		var winner *Session
		for _, s := range sessions {
			if s.Visible && s.Out != nil && (winner == nil || s.Interaction > winner.Interaction) {
				winner = s
			}
		}
		destination, activity := i.Last, "Na plataforma"
		if winner != nil {
			if winner.Destination != "platform" {
				i.Last = winner.Destination
				destination = i.Last
				activity = Activity(destination)
			}
		}
		next[i.ID] = Person{ID: i.ID, Name: labels[user], Destination: destination, Activity: activity}
	}
	if reflect.DeepEqual(next, h.current) {
		return
	}
	changed := []Person{}
	removed := []string{}
	for id, p := range next {
		if old, ok := h.current[id]; !ok || old != p {
			changed = append(changed, p)
		}
	}
	for id := range h.current {
		if _, ok := next[id]; !ok {
			removed = append(removed, id)
		}
	}
	h.current = next
	h.version++
	event := Event{Type: "update", Version: h.version, Participants: changed, Removed: removed}
	for _, s := range h.sessions {
		if s.Out != nil {
			select {
			case s.Out <- event:
			default: // Slow readers get a current snapshot instead of stale incremental events.
				for len(s.Out) > 0 {
					<-s.Out
				}
				s.Out <- Event{Type: "snapshot", Version: h.version, Self: h.users[s.User].ID, Participants: h.people()}
			}
		}
	}
}
func presentationNumber(n int) string { // Short ordinal, never a database identifier.
	const digits = "0123456789"
	if n < 10 {
		return string(digits[n])
	}
	return presentationNumber(n/10) + string(digits[n%10])
}

// Resolve only the opaque identities already visible to an authenticated participant.
func (h *Hub) ResolveParticipant(id string) (uint, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for user, i := range h.users {
		if i.ID == id {
			_, ok := h.current[id]
			return user, ok
		}
	}
	return 0, false
}
func (h *Hub) PersonForUser(user uint) (Person, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	i := h.users[user]
	if i == nil {
		return Person{}, false
	}
	p, ok := h.current[i.ID]
	return p, ok
}

// Private notifications never enter publish(), which broadcasts public presence.
// Notifications contain a conversation ID only; message bodies stay in the private HTTP API.
func (h *Hub) NotifyConversation(a, b uint, id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range h.sessions {
		if s.Out != nil && (s.User == a || s.User == b) {
			select {
			case s.Out <- Event{Type: "chat", ConversationID: id}:
			default:
			}
		}
	}
}
