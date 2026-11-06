package services

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ClueIntegration is the versioned payload of an `onlife` Quill block.
// Unknown versions/providers remain opaque to readers, never executable HTML.
type ClueIntegration struct {
	Version      int    `json:"v"`
	Provider     string `json:"provider"`
	URL          string `json:"url"`
	Title        string `json:"title,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

var platformNodePath = regexp.MustCompile(`^/nodes/graph\?focus=[1-9][0-9]{0,14}$`)

var platformAgentPath = regexp.MustCompile(`^(/space/agents/[1-9][0-9]{0,14}|/chat_templates/chat\?id=[1-9][0-9]{0,14})$`)

var integrationPath = regexp.MustCompile(`^[A-Za-z0-9_/-]+$`)
var opaqueEmbedKey = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)

func DecodeClueIntegration(value interface{}) (ClueIntegration, error) {
	var raw []byte
	if text, ok := value.(string); ok {
		raw = []byte(text)
	} else {
		raw, _ = json.Marshal(value)
	}
	var block ClueIntegration
	if len(raw) > 16384 || json.Unmarshal(raw, &block) != nil || block.Version < 1 || block.Provider == "" || len(block.Provider) > 50 || len(block.URL) > 2000 || utf8.RuneCountInString(block.Title) > 300 || utf8.RuneCountInString(block.Instructions) > 3000 {
		return block, errors.New("bloco de integração inválido")
	}
	return block, nil
}

// ValidateIntegrationURL validates a reference without contacting its provider.
// Provider outages and embedding restrictions must not prevent saving a clue.
func ValidateIntegrationURL(provider, raw string) error {
	if provider == "node" {
		if !platformNodePath.MatchString(raw) {
			return errors.New("referência de problemática inválida")
		}
		return nil
	}
	if provider == "agent" {
		if !platformAgentPath.MatchString(raw) {
			return errors.New("referência de agente inválida")
		}
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || raw != strings.TrimSpace(raw) || strings.ContainsAny(raw, "\r\n\t<>\\") || len(raw) > 2000 || u.User != nil || u.Hostname() == "" {
		return errors.New("endereço de integração inválido")
	}
	host := strings.ToLower(u.Hostname())
	if provider == "overte" || provider == "vircadia" {
		// The documented desktop clients share the hifi address protocol. No
		// arbitrary HTTPS URL is treated as a compatible world or web client.
		if u.Scheme != "hifi" || host == "localhost" || strings.HasSuffix(host, ".local") || u.RawQuery != "" || u.Fragment != "" {
			return errors.New("use o endereço hifi:// do cliente instalado")
		}
		if ip := net.ParseIP(host); ip != nil && !publicIP(ip) {
			return errors.New("endereço privado não permitido")
		}
		return nil
	}
	if u.Scheme != "https" || (u.Port() != "" && u.Port() != "443") {
		return errors.New("a integração exige HTTPS")
	}
	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")
	if path == "" || !integrationPath.MatchString(path) || strings.Contains(path, "//") || strings.Contains(path, "..") {
		return errors.New("link de conteúdo não reconhecido")
	}
	switch provider {
	case "genially":
		if host != "view.genially.com" || len(parts[0]) != 24 || !regexp.MustCompile(`^[a-fA-F0-9]{24}$`).MatchString(parts[0]) {
			return errors.New("use o link de visualização do Genially")
		}
	case "padlet":
		if host != "padlet.com" || len(parts) != 2 || parts[0] == "api" || len(parts[1]) < 3 {
			return errors.New("use o link do mural Padlet ou de incorporação")
		}
	case "framevr":
		if host != "framevr.io" || len(parts) != 1 {
			return errors.New("use o link de uma sala em framevr.io")
		}
	default:
		return errors.New("provedor indisponível")
	}
	return nil
}

func ValidateClueIntegration(value interface{}) error {
	block, err := DecodeClueIntegration(value)
	if err != nil {
		return err
	}
	// Preserve future content without granting it any rendering capability.
	if block.Version != 1 || !isKnownIntegration(block.Provider) {
		return nil
	}
	return ValidateIntegrationURL(block.Provider, block.URL)
}

func isKnownIntegration(provider string) bool {
	return provider == "node" || provider == "agent" || provider == "genially" || provider == "padlet" || provider == "framevr" || provider == "overte" || provider == "vircadia"
}

func HasExtendedQuillBlocks(value string) bool {
	var ops []map[string]interface{}
	if json.Unmarshal([]byte(value), &ops) != nil {
		return false
	}
	for _, op := range ops {
		if embed, ok := op["insert"].(map[string]interface{}); ok {
			if _, video := embed["video"]; !video {
				return true
			}
		}
	}
	return false
}
