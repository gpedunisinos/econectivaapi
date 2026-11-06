package config

import (
	"os"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestOptionalLocationMigratesLegacyColumns(t *testing.T) {
	dsn := os.Getenv("RIEONLIFE_MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("requires a disposable MySQL database ending in _migration_test")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.Raw("SELECT DATABASE()").Scan(&name).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(name, "_migration_test") {
		t.Fatal("refusing to use a non-test database")
	}
	if db.Migrator().HasTable("living_clues") {
		t.Fatal("requires an empty test database")
	}
	if err := db.Exec("CREATE TABLE living_clues (id bigint unsigned PRIMARY KEY, latitude double NOT NULL, longitude double NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE living_clues") })
	if err := db.Exec("INSERT INTO living_clues VALUES (1, -29.794, -51.151)").Error; err != nil {
		t.Fatal(err)
	}
	migrateLivingClueOptionalLocation(db)
	migrateLivingClueOptionalLocation(db) // Repeated startup must remain safe.
	if err := db.Exec("INSERT INTO living_clues VALUES (2, NULL, NULL)").Error; err != nil {
		t.Fatal(err)
	}
	var original struct{ Latitude, Longitude float64 }
	if err := db.Raw("SELECT latitude, longitude FROM living_clues WHERE id=1").Scan(&original).Error; err != nil {
		t.Fatal(err)
	}
	if original.Latitude != -29.794 || original.Longitude != -51.151 {
		t.Fatal("migration changed existing coordinates")
	}
}

func TestRetiredGeographyPreservesCluesAndGeoContext(t *testing.T) {
	dsn := os.Getenv("RIEONLIFE_MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("requires disposable MySQL database ending in _migration_test")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.Raw("SELECT DATABASE()").Scan(&name).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(name, "_migration_test") {
		t.Fatal("refusing non-test database")
	}
	for _, table := range []string{"living_clues", "biomes", "territories", "environmental_contexts"} {
		if db.Migrator().HasTable(table) {
			t.Fatal("requires empty test database")
		}
	}
	queries := []string{
		"CREATE TABLE biomes (id bigint unsigned PRIMARY KEY)",
		"CREATE TABLE territories (id bigint unsigned PRIMARY KEY, biome_id bigint unsigned, FOREIGN KEY (biome_id) REFERENCES biomes(id))",
		"CREATE TABLE living_clues (id bigint unsigned PRIMARY KEY, name varchar(255), biome_id bigint unsigned, territory_id bigint unsigned, FOREIGN KEY (biome_id) REFERENCES biomes(id), FOREIGN KEY (territory_id) REFERENCES territories(id))",
		"CREATE TABLE environmental_contexts (id bigint unsigned PRIMARY KEY, living_clue_id bigint unsigned, biome varchar(100), FOREIGN KEY (living_clue_id) REFERENCES living_clues(id))",
		"INSERT INTO biomes VALUES (1)", "INSERT INTO territories VALUES (1,1)",
		"INSERT INTO living_clues VALUES (1,'Preservada',1,1)",
		"INSERT INTO environmental_contexts VALUES (1,1,'Mata Atlântica')",
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS environmental_contexts")
		db.Exec("DROP TABLE IF EXISTS living_clues")
		db.Exec("DROP TABLE IF EXISTS territories")
		db.Exec("DROP TABLE IF EXISTS biomes")
	})
	for _, query := range queries {
		if err := db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	migrateRetiredGeography(db)
	migrateRetiredGeography(db)
	for _, table := range []string{"biomes", "territories"} {
		if db.Migrator().HasTable(table) {
			t.Fatal("legacy table remains: " + table)
		}
	}
	for _, column := range []string{"biome_id", "territory_id"} {
		if db.Migrator().HasColumn("living_clues", column) {
			t.Fatal("legacy relationship remains: " + column)
		}
	}
	var clueName, biome string
	if err := db.Raw("SELECT name FROM living_clues WHERE id=1").Scan(&clueName).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Raw("SELECT biome FROM environmental_contexts WHERE living_clue_id=1").Scan(&biome).Error; err != nil {
		t.Fatal(err)
	}
	if clueName != "Preservada" || biome != "Mata Atlântica" {
		t.Fatal("lost clue or Geo API context")
	}
}
