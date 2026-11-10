package cdig

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

// Explicit opt-in: read a live source but write seed fixtures ONLY to an isolated
// local database. The synchronizer itself runs in read-only simulation mode.
func TestLiveCDIGDryRun(t *testing.T) {
	if os.Getenv("CDIG_LIVE_VALIDATE") != "true" {
		t.Skip("live source validation is opt-in")
	}
	if !strings.HasPrefix(os.Getenv("DB_NAME"), "onlife_validation_") || os.Getenv("DB_HOST") != "127.0.0.1" {
		t.Fatal("live validation requires an isolated local destination")
	}
	originalDir, _ := os.Getwd()
	must(t, os.Chdir("../.."))
	defer os.Chdir(originalDir)
	config.InitDatabase()
	cfg, reason := Load()
	if !cfg.Enabled {
		t.Fatalf("live integration disabled: %s", reason)
	}
	if fixturePath := os.Getenv("CDIG_LIVE_LEGACY_FIXTURES"); fixturePath != "" {
		raw, err := os.ReadFile(fixturePath)
		must(t, err)
		var fixtures []models.LivingClue
		must(t, json.Unmarshal(raw, &fixtures))
		category := models.Category{Name: "Acervo Cidadania Digital", Type: "Pista de Dados", ColorHex: "#21B6A8"}
		must(t, config.DB.Create(&category).Error)
		for _, fixture := range fixtures {
			deleted := fixture.DeletedAt.Valid
			fixture.DeletedAt.Valid = false
			fixture.CategoryID = category.ID
			fixture.Category = nil
			fixture.Files = nil
			fixture.EnvironmentalContext = nil
			must(t, config.DB.Create(&fixture).Error)
			if deleted {
				must(t, config.DB.Delete(&fixture).Error)
			}
		}
	}
	var before int64
	config.DB.Unscoped().Model(&models.LivingClue{}).Count(&before)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	report, err := (&Sync{Config: cfg, Client: NewClient(cfg), DB: config.DB}).Run(ctx, true)
	must(t, err)
	if report.Committed || !report.DryRun {
		t.Fatal("live validation must never apply")
	}
	var after int64
	config.DB.Unscoped().Model(&models.LivingClue{}).Count(&after)
	if before != after || config.DB.Migrator().HasTable(&models.CDIGMapping{}) {
		t.Fatal("dry run wrote destination data")
	}
	if path := os.Getenv("CDIG_LIVE_REPORT"); path != "" {
		raw, err := json.MarshalIndent(report, "", "  ")
		must(t, err)
		must(t, os.WriteFile(path, raw, 0600))
	}
	t.Logf("CDIG live dry run: %v", report.Counts)
	if os.Getenv("CDIG_LIVE_APPLY_LOCAL") == "true" {
		sync := Sync{Config: cfg, Client: NewClient(cfg), DB: config.DB}
		first, err := sync.Run(ctx, false)
		must(t, err)
		if !first.Committed {
			t.Fatal("local synthetic application failed")
		}
		second, err := sync.Run(ctx, false)
		must(t, err)
		if second.Counts["created"] != 0 || second.Counts["updated"] != 0 || second.Counts["conflict"] != 0 {
			t.Fatalf("live catalog is not idempotent: %v", second.Counts)
		}
		t.Logf("CDIG local application: %v; repeat: %v", first.Counts, second.Counts)
	}

}
