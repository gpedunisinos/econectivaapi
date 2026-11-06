package services

import (
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var documentURLPattern = regexp.MustCompile(`https?://[^\s<>"\x{FFFC}]+`)

func normalizedDocumentURL(raw string) string {
	u, err := url.Parse(strings.TrimRight(raw, ".,;!?)"))
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	if id := YouTubeID(u.String()); id != "" {
		return "https://www.youtube.com/watch?v=" + id
	}
	u.Fragment = ""
	return u.String()
}
func QuillPreviewURLs(delta, plain string) []string {
	urls := map[string]bool{}
	videos := map[string]bool{}
	add := func(raw string) {
		if value := normalizedDocumentURL(raw); value != "" {
			urls[value] = true
		}
	}
	var ops []map[string]interface{}
	if delta == "" {
		for _, value := range documentURLPattern.FindAllString(plain, -1) {
			add(value)
		}
	} else if json.Unmarshal([]byte(delta), &ops) == nil {
		for _, op := range ops {
			if text, ok := op["insert"].(string); ok {
				for _, value := range documentURLPattern.FindAllString(text, -1) {
					add(value)
				}
			}
			if attrs, ok := op["attributes"].(map[string]interface{}); ok {
				if link, ok := attrs["link"].(string); ok {
					add(link)
				}
			}
			if embed, ok := op["insert"].(map[string]interface{}); ok {
				if video, ok := embed["video"].(string); ok {
					videos[normalizedDocumentURL(video)] = true
				}
				if integration, exists := embed["onlife"]; exists {
					if block, err := DecodeClueIntegration(integration); err == nil {
						videos[normalizedDocumentURL(block.URL)] = true
					}
				}
			}
		}
	}
	out := []string{}
	for value := range urls {
		if !videos[value] {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
