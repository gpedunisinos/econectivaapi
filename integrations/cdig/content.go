package cdig

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/OdairPianta/unisinos_usp_api/models"
	"golang.org/x/net/html"
)

type Item struct {
	Kind, ID, Name, Description, Image, URL, Hash string
	Lat, Lng                                      *float64
	Data                                          models.JSONMap
	Members                                       []Video
}

func digest(v any) string { b, _ := json.Marshal(v); return fmt.Sprintf("%x", sha256.Sum256(b)) }
func plain(s string) string {
	node, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return strings.TrimSpace(s)
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for x := n.FirstChild; x != nil; x = x.NextSibling {
			walk(x)
		}
		if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "br" || n.Data == "div" || n.Data == "li") {
			b.WriteByte('\n')
		}
	}
	walk(node)
	return strings.TrimSpace(b.String())
}
func safeURL(s string) string {
	u, e := url.Parse(strings.TrimSpace(s))
	if e != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
		return ""
	}
	return u.String()
}
func canonical(s string) string {
	u, e := url.Parse(safeURL(s))
	if e != nil || u.Host == "" {
		return ""
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	if strings.Contains(u.Host, "spotify.com") {
		u.RawQuery = ""
	}
	q := u.Query()
	for k := range q {
		if strings.HasPrefix(k, "utm_") {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	return strings.TrimSuffix(u.String(), "/")
}

var youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func youtube(s string) string {
	if youtubeID.MatchString(s) {
		return "https://www.youtube.com/watch?v=" + s
	}
	return ""
}
func (c Config) Items(catalog Catalog) ([]Item, error) {
	items := []Item{}
	add := func(item Item) error {
		item.Name = plain(item.Name)
		if item.ID == "" || len(item.ID) > 64 || item.Name == "" {
			return fmt.Errorf("invalid CDIG %s identity or title", item.Kind)
		}
		if len([]rune(item.Name)) > 255 {
			item.Data["original_title"] = item.Name
			item.Name = string([]rune(item.Name)[:254]) + "…"
		}
		item.Data["source"] = c.SourceID()
		item.Data["environment"] = c.SourceKey
		item.Data["type"] = item.Kind
		item.Data["id"] = item.ID
		item.Data["state"] = "available"
		item.Data["url"] = item.URL
		item.Description = plain(item.Description) + "\n\nFonte: Cidadania Digital\n" + item.URL
		if author, ok := item.Data["author"].(string); ok && author != "" {
			item.Description += "\nAutoria: " + author
		}
		if rights, ok := item.Data["rights"].(string); ok && rights != "" {
			item.Description += "\nDireitos: " + rights
		}
		item.Image = safeURL(item.Image)
		if len(item.Image) > 255 {
			item.Image = ""
		}
		item.Hash = digest(item)
		items = append(items, item)
		return nil
	}
	for _, r := range catalog.Resources {
		names := []string{}
		for _, a := range r.Authors {
			names = append(names, a.Name)
		}
		author := strings.Join(names, "; ")
		if author == "" {
			author = r.Author
		}
		data := models.JSONMap{"author": author, "authors": r.Authors, "publisher": r.Publisher, "rights": r.CopyRights, "language": r.Language, "subject": r.Subject, "tags": r.Tags, "coverage": r.Coverage, "contributor": r.Contributor, "identifier": r.Identifier, "format": r.Format, "format_id": r.FormatID, "original_date": r.OriginalDate, "published_at": r.PublishedAt}
		if err := add(Item{Kind: "resource", ID: r.ID, Name: r.Title, Description: r.Description, URL: safeURL(r.SourceURL), Data: data}); err != nil {
			return nil, err
		}
	}
	for _, v := range catalog.Videos {
		title := v.Title
		if title == "" {
			title = v.SourceTitle
		}
		if title == "" {
			title = "Vídeo CDIG " + v.ID
		}
		u := youtube(v.VideoID)
		if u == "" {
			return nil, fmt.Errorf("invalid CDIG video %s YouTube identity", v.ID)
		}
		image := "https://i.ytimg.com/vi/" + v.VideoID + "/hqdefault.jpg"
		if err := add(Item{Kind: "video", ID: v.ID, Name: title, Description: v.Description, Image: image, URL: u, Data: models.JSONMap{"author": v.SourceAuthor, "channel": v.SourceChannel, "year": v.SourceYear, "subject": v.Subject}}); err != nil {
			return nil, err
		}
	}
	for _, s := range catalog.Series {
		if err := add(Item{Kind: "serie", ID: s.ID, Name: s.Title, Description: s.Description, Image: s.Thumbnail, URL: c.BaseURL + "/series/" + url.PathEscape(s.Slug), Members: s.Videos, Data: models.JSONMap{"slug": s.Slug, "episodes": s.Videos}}); err != nil {
			return nil, err
		}
	}
	for _, p := range catalog.Projects {
		if p.Lat == nil || p.Lng == nil || *p.Lat < -90 || *p.Lat > 90 || *p.Lng < -180 || *p.Lng > 180 {
			return nil, fmt.Errorf("invalid CDIG map project %s coordinates", p.ID)
		}
		if err := add(Item{Kind: "map_project", ID: p.ID, Name: p.Name, Description: p.Description, URL: c.BaseURL + "/mapa", Lat: p.Lat, Lng: p.Lng, Data: models.JSONMap{"country": p.Country, "city": p.City, "proposers": p.Proposers, "nature": p.Nature, "purpose": p.Purpose, "themes": p.Themes, "tics": p.Tics, "website": safeURL(p.Website), "app": safeURL(p.App), "researcher": p.Researcher}}); err != nil {
			return nil, err
		}
	}
	return items, nil
}
func delta(text string) string {
	b, _ := json.Marshal([]map[string]any{{"insert": text + "\n"}})
	return string(b)
}
func ownedHash(clue models.LivingClue) string {
	fields := []any{clue.Name, clue.Description, clue.DescriptionDelta, clue.Latitude, clue.Longitude, clue.ImageURL, ownedMetadata(clue.Metadata["cdig"])}
	if m, ok := clue.Metadata["cdig"].(map[string]any); ok && m["type"] == "serie" {
		fields = append(fields, clue.Metadata["colecao_ordem_itens_v1"])
	}
	if m, ok := clue.Metadata["cdig"].(models.JSONMap); ok && m["type"] == "serie" {
		fields = append(fields, clue.Metadata["colecao_ordem_itens_v1"])
	}
	return digest(fields)
}

// Availability is operational state, not editorial content or a local edit.
func ownedMetadata(value any) any {
	raw, _ := json.Marshal(value)
	var data map[string]any
	_ = json.Unmarshal(raw, &data)
	delete(data, "state")
	return data
}
