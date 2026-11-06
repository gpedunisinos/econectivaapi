package tests

import (
	"reflect"
	"strings"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/models"
)

func TestCategoryMetadataTagsDatabaseConstraints(t *testing.T) {
	field, found := reflect.TypeOf(models.Category{}).FieldByName("MetadataTags")
	if !found {
		t.Fatal("MetadataTags field not found")
	}

	gormTag := field.Tag.Get("gorm")
	if !strings.Contains(gormTag, "type:text") {
		t.Errorf("MetadataTags must remain a text column, got %q", gormTag)
	}
	if !strings.Contains(gormTag, "notNull") {
		t.Errorf("MetadataTags must remain NOT NULL, got %q", gormTag)
	}
	if strings.Contains(strings.ToLower(gormTag), "default") {
		t.Errorf("MetadataTags must not define a default for MySQL 8 compatibility, got %q", gormTag)
	}
}
