package services

import (
	"regexp"
	"strings"

	"github.com/OdairPianta/unisinos_usp_api/models"
)

var geminiCredentialPattern = regexp.MustCompile(`AIza[0-9A-Za-z_-]{20,}`)

func sensitiveAIField(key string) bool {
	key = strings.ToLower(key)
	for _, part := range []string{"password", "senha", "secret", "token", "api_key", "apikey", "authorization", "credential", "credencial"} {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

func SanitizeAIText(value string, limit int) string {
	value = geminiCredentialPattern.ReplaceAllString(value, "[credencial omitida]")
	value = documentURLPattern.ReplaceAllStringFunc(value, func(raw string) string {
		if _, err := SafeSourceURL(strings.TrimRight(raw, ".,;!?)")); err != nil {
			return "[endereço não público ou com credenciais omitido]"
		}
		return raw
	})
	return boundedAIText(value, limit)
}

// SanitizeAIData limits arbitrary form/metadata values without serializing
// authentication or administrative configuration fields into model context.
func SanitizeAIData(value interface{}, depth int) interface{} {
	if depth > 4 {
		return "[limite de profundidade]"
	}
	switch v := value.(type) {
	case map[string]interface{}:
		out := map[string]interface{}{}
		count := 0
		for key, item := range v {
			if sensitiveAIField(key) {
				continue
			}
			if count >= 30 {
				out["_limitation"] = "Quantidade de campos limitada."
				break
			}
			out[boundedAIText(key, 100)] = SanitizeAIData(item, depth+1)
			count++
		}
		return out
	case models.JSONMap:
		return SanitizeAIData(map[string]interface{}(v), depth)
	case []interface{}:
		out := []interface{}{}
		for i, item := range v {
			if i >= 20 {
				break
			}
			out = append(out, SanitizeAIData(item, depth+1))
		}
		return out
	case string:
		return SanitizeAIText(v, 2000)
	default:
		return value
	}
}
