package cdig

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

type Subject struct{ ID, Name, Slug string }
type Author struct{ ID, Name string }
type Tag struct{ ID, Name string }
type Resource struct {
	ID, Title, Description, Author, Publisher, Language, Coverage, Contributor, Identifier, Format string
	SourceURL                                                                                      string  `json:"source_url"`
	PublishedAt                                                                                    *string `json:"published_at"`
	IsPublished                                                                                    bool    `json:"is_published"`
	CopyRights                                                                                     string  `json:"copy_rights"`
	OriginalDate                                                                                   string  `json:"original_date"`
	FormatID                                                                                       int     `json:"format_id"`
	Authors                                                                                        []Author
	Tags                                                                                           []Tag
	Subject                                                                                        *Subject
}
type Episode struct {
	Title, Description string
	Position           int
}
type Video struct {
	ID, Title, Description string
	VideoID                string `json:"video_id"`
	IsActive               bool   `json:"is_active"`
	SourceAuthor           string `json:"source_author"`
	SourceTitle            string `json:"source_title"`
	SourceChannel          string `json:"source_channel"`
	SourceYear             *int   `json:"source_year"`
	Subject                *Subject
	Episode                *Episode
}
type Serie struct {
	ID, Slug, Title, Description, Thumbnail string
	IsActive                                bool `json:"is_active"`
	Videos                                  []Video
}
type MapProject struct {
	ID, Name, Country, City, Proposers, Nature, Purpose, Description, Website, Instagram, Facebook, Tiktok, App, Researcher string
	Lat, Lng                                                                                                                *float64
	Themes, Tics                                                                                                            []string
}
type Catalog struct {
	Resources []Resource
	Videos    []Video
	Series    []Serie
	Projects  []MapProject
}
type pageInfo struct {
	Total, CurrentPage int
	HasMorePages       bool
}
type page[T any] struct {
	Data          []T
	PaginatorInfo pageInfo
}

type Client struct {
	Config Config
	HTTP   *http.Client
}

func NewClient(c Config) *Client {
	return &Client{Config: c, HTTP: &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("CDIG redirects are not allowed") }}}
}

// Errors contain status and operation only, never headers, response bodies,
// credentials or URLs returned by an untrusted remote error.
func (c *Client) request(ctx context.Context, method, path, token string, input, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return errors.New("CDIG request encoding failed")
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Config.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return errors.New("CDIG request creation failed")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return errors.New("CDIG transport failed")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("CDIG %s returned HTTP %d", path, res.StatusCode)
	}
	const max = 16 << 20
	body, err := io.ReadAll(io.LimitReader(res.Body, max+1))
	if err != nil || len(body) > max {
		return errors.New("CDIG response is too large or unreadable")
	}
	if output != nil && json.Unmarshal(body, output) != nil {
		return errors.New("CDIG response is invalid JSON")
	}
	return nil
}
func (c *Client) withToken(ctx context.Context, run func(string) error) (err error) {
	token := c.Config.Token
	if token != "" {
		return run(token)
	}
	var issued struct {
		Token string
		Scope string
	}
	if err = c.request(ctx, "POST", "/api/integration/token", "", map[string]any{"email": c.Config.Email, "password": c.Config.Password, "name": "USP Atopos content synchronization", "expires_in_days": 1}, &issued); err != nil {
		return err
	}
	if issued.Token == "" {
		return errors.New("CDIG did not issue an integration token")
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		e := c.request(cleanup, "DELETE", "/api/integration/token", issued.Token, nil, nil)
		if err == nil && e != nil {
			err = errors.New("CDIG temporary token revocation failed")
		}
	}()
	if issued.Scope != "content:read" {
		return errors.New("CDIG issued an unexpected token scope")
	}
	return run(issued.Token)
}
func fetchPages[T any](ctx context.Context, c *Client, token, root, fields string, filter string) ([]T, error) {
	items := []T{}
	seen := map[string]bool{}
	total := -1
	for number := 1; number <= 10000; number++ {
		query := fmt.Sprintf("{ %s(first:100,page:%d%s) { data { %s } paginatorInfo { total currentPage hasMorePages } } }", root, number, filter, fields)
		var response struct {
			Data   map[string]json.RawMessage
			Errors json.RawMessage
		}
		if err := c.request(ctx, "POST", "/api/graphql", token, map[string]any{"query": query}, &response); err != nil {
			return nil, err
		}
		if len(response.Errors) > 0 && string(response.Errors) != "null" {
			return nil, fmt.Errorf("CDIG rejected %s query", root)
		}
		var p page[T]
		if response.Data[root] == nil || json.Unmarshal(response.Data[root], &p) != nil {
			return nil, errors.New("CDIG returned an invalid page")
		}
		if p.PaginatorInfo.CurrentPage != number || p.PaginatorInfo.Total < 0 {
			return nil, errors.New("CDIG pagination is inconsistent")
		}
		if total == -1 {
			total = p.PaginatorInfo.Total
		}
		if total != p.PaginatorInfo.Total {
			return nil, errors.New("CDIG catalog changed during pagination; retry")
		}
		for _, item := range p.Data {
			raw, _ := json.Marshal(item)
			var identity struct{ ID string }
			_ = json.Unmarshal(raw, &identity)
			if identity.ID == "" || seen[identity.ID] {
				return nil, errors.New("CDIG returned a missing or duplicate identity")
			}
			seen[identity.ID] = true
			items = append(items, item)
		}
		if !p.PaginatorInfo.HasMorePages {
			if len(items) != total {
				return nil, errors.New("CDIG pagination is incomplete")
			}
			return items, nil
		}
		if len(p.Data) == 0 {
			return nil, errors.New("CDIG returned an empty intermediate page")
		}
	}
	return nil, errors.New("CDIG pagination limit exceeded")
}
func (c *Client) Fetch(ctx context.Context) (catalog Catalog, err error) {
	if !c.Config.Enabled {
		return catalog, nil
	}
	err = c.withToken(ctx, func(token string) error {
		var e error
		catalog.Resources, e = fetchPages[Resource](ctx, c, token, "resources", `id title description published_at is_published author authors { id name } publisher language coverage contributor identifier format format_id copy_rights original_date source_url subject { id name slug } tags { id name }`, ",is_published:true")
		if e != nil {
			return e
		}
		catalog.Videos, e = fetchPages[Video](ctx, c, token, "videos", `id video_id title description is_active source_author source_title source_channel source_year subject { id name slug }`, ",is_active:true")
		if e != nil {
			return e
		}
		catalog.Series, e = fetchPages[Serie](ctx, c, token, "series", `id slug title description thumbnail is_active videos(is_active:true) { id video_id title description is_active episode { title description position } }`, ",is_active:true")
		if e != nil {
			return e
		}
		catalog.Projects, e = fetchPages[MapProject](ctx, c, token, "mapProjects", `id name country city lat lng proposers nature purpose description themes tics website instagram facebook tiktok app researcher`, "")
		if e != nil {
			return e
		}
		for _, r := range catalog.Resources {
			if !r.IsPublished || r.PublishedAt == nil {
				return errors.New("CDIG returned an unpublished resource")
			}
		}
		videoIDs := map[string]bool{}
		for _, v := range catalog.Videos {
			if !v.IsActive {
				return errors.New("CDIG returned an inactive video")
			}
			videoIDs[v.ID] = true
		}
		for i := range catalog.Series {
			s := &catalog.Series[i]
			if !s.IsActive {
				return errors.New("CDIG returned an inactive series")
			}
			ids := map[string]bool{}
			for _, v := range s.Videos {
				if !v.IsActive || v.Episode == nil || ids[v.ID] || !videoIDs[v.ID] {
					return errors.New("CDIG series contains invalid or missing episodes")
				}
				ids[v.ID] = true
			}
			sort.SliceStable(s.Videos, func(i, j int) bool { return s.Videos[i].Episode.Position < s.Videos[j].Episode.Position })
		}
		return nil
	})
	return
}
