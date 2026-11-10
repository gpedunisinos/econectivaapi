package cdig

import (
	"context"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func TestSyncIsolatedMySQL(t *testing.T) {
	if !strings.HasPrefix(os.Getenv("DB_NAME"), "onlife_validation_") {
		t.Skip("requires synthetic isolated MySQL via scripts/run-isolated-tests.py")
	}
	originalDir, _ := os.Getwd()
	must(t, os.Chdir("../.."))
	defer os.Chdir(originalDir)
	config.InitDatabase()
	db := config.DB
	published := "2026-11-10 12:00:00"
	catalog := Catalog{Resources: []Resource{{ID: "1", Title: "Reference", Description: "<p>Published</p>", SourceURL: "https://example.test/reference", IsPublished: true, PublishedAt: &published}, {ID: "2", Title: "Deleted legacy", SourceURL: "https://example.test/deleted", IsPublished: true, PublishedAt: &published}}, Videos: []Video{{ID: "1", Title: "Video", VideoID: "abcdefghijk", IsActive: true}, {ID: "2", Title: "Second", VideoID: "lmnopqrstuv", IsActive: true}}, Series: []Serie{{ID: "1", Title: "Series", Slug: "series", IsActive: true, Videos: []Video{{ID: "2", IsActive: true, Episode: &Episode{Position: 1}}, {ID: "1", IsActive: true, Episode: &Episode{Position: 2}}}}, {ID: "2", Title: "Shared episode", Slug: "shared", IsActive: true, Videos: []Video{{ID: "1", IsActive: true, Episode: &Episode{Position: 1}}}}}}
	fail := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(503)
			return
		}
		var input struct{ Query string }
		_ = json.NewDecoder(r.Body).Decode(&input)
		var root string
		var data any
		count := 0
		switch {
		case strings.Contains(input.Query, "resources("):
			root = "resources"
			data = catalog.Resources
			count = len(catalog.Resources)
		case strings.Contains(input.Query, "series("):
			root = "series"
			data = catalog.Series
			count = len(catalog.Series)
		case strings.Contains(input.Query, "mapProjects("):
			root = "mapProjects"
			data = catalog.Projects
			count = len(catalog.Projects)
		default:
			root = "videos"
			data = catalog.Videos
			count = len(catalog.Videos)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{root: map[string]any{"data": data, "paginatorInfo": map[string]any{"total": count, "currentPage": 1, "hasMorePages": false}}}})
	}))
	defer server.Close()
	cfg := Config{Enabled: true, BaseURL: server.URL, SourceKey: "isolated", Token: "synthetic", ActorID: 1}
	sync := Sync{Config: cfg, Client: NewClient(cfg), DB: db}
	// A verified prior import is linked, and its presentation is retained.
	category := models.Category{Name: "Acervo Cidadania Digital", Type: "Pista de Dados", ColorHex: "#21B6A8"}
	must(t, db.Create(&category).Error)
	legacy := models.LivingClue{Name: "Legacy presentation", CategoryID: category.ID, Description: "Keep this local presentation", ImageURL: "https://example.test/cover.jpg", Metadata: models.JSONMap{"acervo": "Cidadania Digital", "identificador_origem": "resource-1", "recurso_original": "https://example.test/reference", "local": "preserve"}}
	must(t, db.Create(&legacy).Error)
	deleted := models.LivingClue{Name: "Deleted", CategoryID: category.ID, Metadata: models.JSONMap{"acervo": "Cidadania Digital", "identificador_origem": "resource-2", "recurso_original": "https://example.test/deleted"}}
	must(t, db.Create(&deleted).Error)
	must(t, db.Delete(&deleted).Error)
	report, err := sync.Run(context.Background(), true)
	must(t, err)
	if report.Counts["linked"] != 1 || report.Counts["suppressed"] != 1 || report.Counts["created"] != 4 {
		t.Fatalf("unexpected plan: %+v", report.Counts)
	}
	if db.Migrator().HasTable(&models.CDIGMapping{}) {
		t.Fatal("dry run created tracking tables")
	}
	var count int64
	db.Unscoped().Model(&models.LivingClue{}).Count(&count)
	if count != 2 {
		t.Fatal("dry run wrote clues")
	}
	report, err = sync.Run(context.Background(), false)
	must(t, err)
	must(t, db.First(&legacy, legacy.ID).Error)
	if legacy.Name != "Legacy presentation" || legacy.Metadata["local"] != "preserve" {
		t.Fatal("reconciliation overwrote local data")
	}
	// Repetition and series membership never create a second video clue.
	report, err = sync.Run(context.Background(), false)
	must(t, err)
	if report.Counts["created"] != 0 || report.Counts["unchanged"] != 5 || report.Counts["suppressed"] != 1 {
		t.Fatalf("not idempotent: %+v", report.Counts)
	}
	db.Model(&models.ClueConnection{}).Count(&count)
	if count != 3 {
		t.Fatalf("expected 3 series memberships, got %d", count)
	}
	// An editorial change updates a pristine imported clue, preserving local metadata.
	var mapping models.CDIGMapping
	must(t, db.Where("kind = ? AND external_id = ?", "video", "1").First(&mapping).Error)
	var video models.LivingClue
	must(t, db.First(&video, mapping.ClueID).Error)
	video.Metadata["local_note"] = "keep"
	must(t, db.Model(&video).Update("metadata", video.Metadata).Error)
	catalog.Videos[0].Title = "Updated source title"
	report, err = sync.Run(context.Background(), false)
	must(t, err)
	must(t, db.First(&video, video.ID).Error)
	if video.Name != "Updated source title" || video.Metadata["local_note"] != "keep" {
		t.Fatal("source update lost local metadata")
	}
	// Locally edited managed fields are reported rather than overwritten.
	must(t, db.Model(&video).Update("name", "Manual edit").Error)
	catalog.Videos[0].Title = "Second source change"
	report, err = sync.Run(context.Background(), false)
	must(t, err)
	if report.Counts["conflict"] != 1 {
		t.Fatalf("missing local conflict: %v", report.Counts)
	}
	must(t, db.First(&video, video.ID).Error)
	if video.Name != "Manual edit" {
		t.Fatal("local edit overwritten")
	}
	// Series reorder/remove affects only importer-owned links. Shared videos and
	// a locally added member survive, with local members appended to source order.
	var seriesMapping models.CDIGMapping
	must(t, db.Where("kind = ? AND external_id = ?", "serie", "1").First(&seriesMapping).Error)
	var relation models.ConnectionType
	must(t, db.Where("name = ?", "Coleção contém item").First(&relation).Error)
	local := models.LivingClue{Name: "Local member", CategoryID: category.ID}
	must(t, db.Create(&local).Error)
	must(t, db.Create(&models.ClueConnection{SourceClueID: seriesMapping.ClueID, TargetClueID: local.ID, ConnectionTypeID: relation.ID}).Error)
	// Use the unchanged second video: the first deliberately has a local conflict.
	catalog.Series[0].Videos = []Video{{ID: "2", IsActive: true, Episode: &Episode{Position: 1}}}
	report, err = sync.Run(context.Background(), false)
	must(t, err)
	var collection models.LivingClue
	must(t, db.First(&collection, seriesMapping.ClueID).Error)
	order := collection.Metadata["colecao_ordem_itens_v1"].([]any)
	if len(order) != 2 || uint(order[1].(float64)) != local.ID {
		t.Fatal("local membership/order was lost")
	}
	var videoCount int64
	db.Model(&models.LivingClue{}).Where("id = ?", video.ID).Count(&videoCount)
	if videoCount != 1 {
		t.Fatal("removing from a series deleted its independent clue")
	}
	// A failed snapshot writes nothing and never marks all records as unavailable.
	fail = true
	_, err = sync.Run(context.Background(), false)
	if err == nil {
		t.Fatal("source failure ignored")
	}
	db.Model(&models.CDIGMapping{}).Where("missing = ?", true).Count(&count)
	if count != 0 {
		t.Fatal("source failure treated as removal")
	}
	fail = false
	// A complete snapshot can mark absence without deleting the clue or history.
	catalog.Resources = nil
	report, err = sync.Run(context.Background(), false)
	must(t, err)
	if report.Counts["unavailable"] != 1 {
		t.Fatalf("missing absence marker: %+v", report.Counts)
	}
	must(t, db.First(&legacy, legacy.ID).Error)
	catalog.Resources = []Resource{{ID: "1", Title: "Reference", Description: "<p>Published</p>", SourceURL: "https://example.test/reference", IsPublished: true, PublishedAt: &published}}
	report, err = sync.Run(context.Background(), false)
	must(t, err)
	if report.Counts["restored"] != 1 {
		t.Fatal("source availability was not restored")
	}
	must(t, db.First(&legacy, legacy.ID).Error)
	if legacy.Description != "Keep this local presentation" {
		t.Fatal("restoring availability overwrote adopted legacy content")
	}
	// Advisory lock contention prevents a competing writer.
	must(t, db.Connection(func(connDB *gorm.DB) error {
		var locked int
		key := "cdig:" + cfg.SourceID()[:48]
		must(t, connDB.Raw("SELECT GET_LOCK(?,0)", key).Scan(&locked).Error)
		defer connDB.Exec("SELECT RELEASE_LOCK(?)", key)
		_, e := sync.Run(context.Background(), false)
		if e == nil {
			return fmt.Errorf("concurrent run was accepted")
		}
		return nil
	}))
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
