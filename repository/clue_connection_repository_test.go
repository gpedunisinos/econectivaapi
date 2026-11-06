package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strings"
	"testing"
)

func TestClueConnectionUpdateIncludesZeroAndEmptyEvidence(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: "tcp(127.0.0.1:1)/unused", SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	previous := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previous })
	var statement string
	var values []interface{}
	if err := db.Callback().Update().After("gorm:update").Register("capture_connection_values", func(tx *gorm.DB) { statement = tx.Statement.SQL.String(); values = tx.Statement.Vars }); err != nil {
		t.Fatal(err)
	}
	original := &models.ClueConnection{DefaultModel: models.DefaultModel{ID: 7}, SourceClueID: 1, TargetClueID: 2, ConnectionTypeID: 3, Strength: 5, Evidence: "Old"}
	changed := &models.ClueConnection{SourceClueID: 1, TargetClueID: 2, ConnectionTypeID: 3, Strength: 0, Evidence: ""}
	if err := UpdateClueConnection(original, changed); err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"`strength`=?", "`evidence`=?"} {
		if !strings.Contains(statement, column) {
			t.Fatalf("Missing %s in %s", column, statement)
		}
	}
	var zero, empty bool
	for _, value := range values {
		if n, ok := value.(int); ok && n == 0 {
			zero = true
		}
		if s, ok := value.(string); ok && s == "" {
			empty = true
		}
	}
	if !zero || !empty {
		t.Fatal("Explicit zero and empty evidence must be bound")
	}
	for _, forbidden := range []string{"`created_at`=", "`deleted_at`=", "SET `id`="} {
		if strings.Contains(statement, forbidden) {
			t.Fatal("Unrelated state overwritten")
		}
	}
}
