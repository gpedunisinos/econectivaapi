package services

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var ErrInvalidQuillDelta = errors.New("description_delta deve conter um documento Quill válido com texto, YouTube ou integrações reconhecidas")
var youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func YouTubeID(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	id := ""
	switch host {
	case "youtu.be":
		id = strings.Trim(u.Path, "/")
	case "youtube.com", "www.youtube.com", "m.youtube.com", "www.youtube-nocookie.com":
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if u.Path == "/watch" {
			id = u.Query().Get("v")
		} else if len(parts) == 2 && (parts[0] == "embed" || parts[0] == "shorts" || parts[0] == "live") {
			id = parts[1]
		}
	}
	if youtubeID.MatchString(id) {
		return id
	}
	return ""
}
func ValidateQuillDelta(value string) error {
	if value == "" {
		return nil
	}
	var ops []map[string]interface{}
	if json.Unmarshal([]byte(value), &ops) != nil || ops == nil {
		return ErrInvalidQuillDelta
	}
	for _, op := range ops {
		if len(op) == 0 || op["retain"] != nil || op["delete"] != nil {
			return ErrInvalidQuillDelta
		}
		switch insert := op["insert"].(type) {
		case string:
		case map[string]interface{}:
			if len(insert) != 1 {
				return ErrInvalidQuillDelta
			}
			for key, value := range insert {
				switch key {
				case "video":
					v, ok := value.(string)
					if !ok || YouTubeID(v) == "" {
						return ErrInvalidQuillDelta
					}
				case "onlife":
					if ValidateClueIntegration(value) != nil {
						return ErrInvalidQuillDelta
					}
				default:
					// Opaque legacy/future embeds survive editing. The reader uses a
					// neutral placeholder, never supplied scripts or markup.
					data, err := json.Marshal(value)
					if !opaqueEmbedKey.MatchString(key) || err != nil || len(data) > 16384 {
						return ErrInvalidQuillDelta
					}
				}
			}
		default:
			return ErrInvalidQuillDelta
		}
	}
	return nil
}

// Only author text is searchable; embeds do not imply transcripts or fetched-page text.
func QuillPlainText(value string) string {
	var ops []map[string]interface{}
	if json.Unmarshal([]byte(value), &ops) != nil {
		return ""
	}
	var b strings.Builder
	for _, op := range ops {
		if text, ok := op["insert"].(string); ok {
			b.WriteString(text)
		}
	}
	return strings.TrimSpace(b.String())
}
