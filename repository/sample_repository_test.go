package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSampleUpdateIncludesExplicitZeroAndEmptyOptionalField(t *testing.T) {
	// No live database, credentials, migrations or network access are required.
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: "tcp(127.0.0.1:1)/unused?parseTime=true", SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true,
		SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	previous := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previous })
	var statement string
	var values []interface{}
	err = db.Callback().Update().After("gorm:update").Register("capture_sample_update", func(tx *gorm.DB) {
		statement = tx.Statement.SQL.String()
		values = append([]interface{}(nil), tx.Statement.Vars...)
	})
	if err != nil {
		t.Fatal(err)
	}
	original := &models.SampleModel{DefaultModel: models.DefaultModel{ID: 7},
		Name: "Original", SampleNullable: "Optional value", SampleDouble: 5}
	updated := &models.SampleModel{Name: "Edited", SampleString: "Synthetic",
		SampleUnique: "sample-update-regression", SampleDate: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		SampleNullable: "", SampleDouble: 0, SampleDetailID: 2, OrderNumber: 1}
	if err := UpdateSampleModel(original, updated); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(statement, "`sample_nullable`=?") {
		t.Fatalf("The update must include the explicitly cleared optional field: %s", statement)
	}
	if !strings.Contains(statement, "`sample_double`=?") {
		t.Fatal("Explicit numeric zero must be included in the update")
	}
	zeroValue := false
	for _, value := range values {
		if n, ok := value.(float64); ok && n == 0 {
			zeroValue = true
		}
	}
	if !zeroValue {
		t.Fatal("Expected bound numeric zero")
	}
	emptyValue := false
	for _, value := range values {
		if text, ok := value.(string); ok && text == "" {
			emptyValue = true
		}
	}
	if !emptyValue {
		t.Fatal("Expected an empty string among the bound update values")
	}
	for _, forbidden := range []string{"`created_at`=", "`deleted_at`=", "SET `id`="} {
		if strings.Contains(statement, forbidden) {
			t.Fatalf("Update must not overwrite unrelated model state: %s", statement)
		}
	}
}
