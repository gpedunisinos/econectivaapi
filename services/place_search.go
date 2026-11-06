package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var ErrPlaceBusy = errors.New("A busca de lugares está ocupada. Aguarde um instante e tente novamente.")

type placeCache struct {
	body    json.RawMessage
	expires time.Time
}
type PlaceSearch struct {
	mu       sync.Mutex
	next     time.Time
	cache    map[string]placeCache
	Client   *http.Client
	Endpoint string
}

func NewPlaceSearch() *PlaceSearch {

	return &PlaceSearch{cache: make(map[string]placeCache), Client: &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (s *PlaceSearch) Search(ctx context.Context, query string) (json.RawMessage, error) {
	query = strings.Join(strings.Fields(query), " ")
	key := strings.ToLower(query)
	if len([]rune(query)) < 3 || len([]rune(query)) > 200 {
		return nil, errors.New("Informe um lugar com 3 a 200 caracteres.")
	}
	s.mu.Lock()
	now := time.Now()
	if cached, ok := s.cache[key]; ok && now.Before(cached.expires) {
		s.mu.Unlock()
		return cached.body, nil
	}
	if now.Before(s.next) {
		s.mu.Unlock()
		return nil, ErrPlaceBusy
	}
	s.next = now.Add(1100 * time.Millisecond)
	s.mu.Unlock()
	cached, finish, sharedErr := sharedPlaceSearch(key)
	if sharedErr != nil {
		return nil, sharedErr
	}
	if cached != nil {
		return cached, nil
	}
	var sharedResult []byte
	defer func() { finish(sharedResult) }()
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = os.Getenv("PLACE_SEARCH_URL")
	}
	if endpoint == "" {
		endpoint = "https://nominatim.openstreetmap.org/search"
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("format", "jsonv2")
	q.Set("limit", "5")
	q.Set("addressdetails", "1")
	q.Del("countrycodes")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "EducacaoOnLIFE/1.0 (https://educacaoonlife.gpedu.com.br)")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.6")
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New("place provider unavailable")
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024+1))
	if err != nil || len(raw) > 1024*1024 {
		return nil, errors.New("invalid place response")
	}
	var places []map[string]interface{}
	if json.Unmarshal(raw, &places) != nil || places == nil {
		return nil, errors.New("invalid place response")
	}
	s.mu.Lock()
	if len(s.cache) >= 512 {
		s.cache = make(map[string]placeCache)
	}
	s.cache[key] = placeCache{raw, time.Now().Add(24 * time.Hour)}
	sharedResult = raw
	s.mu.Unlock()
	return raw, nil
}
