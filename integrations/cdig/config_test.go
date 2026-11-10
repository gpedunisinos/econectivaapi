package cdig

import (
	"encoding/json"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"testing"
)

func TestConfigurationFailClosed(t *testing.T) {
	values := map[string]string{"CDIG_SYNC_ENABLED": "true", "CDIG_BASE_URL": "https://cdig.example", "CDIG_SOURCE_KEY": "staging", "CDIG_API_TOKEN": "synthetic", "CDIG_IMPORT_USER_ID": "1"}
	get := func(k string) string { return values[k] }
	cfg, _ := FromEnv(get)
	if !cfg.Enabled || cfg.Interval != 0 {
		t.Fatal("complete config should enable manual mode")
	}
	first := cfg.SourceID()
	values["CDIG_SOURCE_KEY"] = "production"
	cfg, _ = FromEnv(get)
	if cfg.SourceID() == first {
		t.Fatal("environment identities overlap")
	}
	values["CDIG_SOURCE_KEY"] = "staging"
	for _, key := range []string{"CDIG_SYNC_ENABLED", "CDIG_BASE_URL", "CDIG_SOURCE_KEY", "CDIG_API_TOKEN", "CDIG_IMPORT_USER_ID"} {
		old := values[key]
		delete(values, key)
		c, _ := FromEnv(get)
		if c.Enabled {
			t.Fatalf("missing %s should disable", key)
		}
		values[key] = old
	}
	for _, bad := range []string{"http://cdig.example", "https://user:secret@cdig.example", "https://cdig.example?token=secret", "https://cdig.example/subpath"} {
		values["CDIG_BASE_URL"] = bad
		c, reason := FromEnv(get)
		if c.Enabled || reason == "" {
			t.Fatal("invalid origin accepted")
		}
	}
	values["CDIG_BASE_URL"] = "https://cdig.example"
	values["CDIG_API_TOKEN"] = ""
	values["CDIG_EMAIL"] = "test@example.test"
	values["CDIG_PASSWORD"] = "synthetic-password"
	cfg, _ = FromEnv(get)
	if !cfg.Enabled {
		t.Fatal("credential mode should enable")
	}
}
func TestCanonicalHashSurvivesDatabaseJSON(t *testing.T) {
	clue := models.LivingClue{Name: "Title", Metadata: models.JSONMap{"cdig": models.JSONMap{"type": "serie", "state": "available", "episodes": []Video{{ID: "1", Title: "Title", Episode: &Episode{Title: "Episode", Position: 1}}}}, "colecao_ordem_itens_v1": []uint{7, 8}}}
	first := ownedHash(clue)
	raw, _ := json.Marshal(clue.Metadata)
	_ = json.Unmarshal(raw, &clue.Metadata)
	if ownedHash(clue) != first {
		t.Fatal("JSON round trip changed fingerprint")
	}
	clue.Metadata["cdig"].(map[string]any)["state"] = "unavailable"
	if ownedHash(clue) != first {
		t.Fatal("availability should not become a local edit")
	}
	clue.Name = "Manual edit"
	if ownedHash(clue) == first {
		t.Fatal("local changes must be detected")
	}
}
