package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDatabase() {
	if DB != nil {
		return
	}
	err := godotenv.Load(".env")

	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("Error loading .env file")
	}

	DbHost := os.Getenv("DB_HOST")
	DbUser := os.Getenv("DB_USER")
	DbPassword := os.Getenv("DB_PASSWORD")
	DbName := os.Getenv("DB_NAME")
	DbPort := os.Getenv("DB_PORT")
	if expected := os.Getenv("EXPECTED_DATABASE_NAME"); expected != "" && DbName != expected {
		panic("Database does not match this platform deployment")
	}

	dsn := buildDatabaseDSN(DbUser, DbPassword, DbHost, DbPort, DbName)

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})

	if err != nil {
		sentry.CaptureException(err)
		panic("Failed to connect to database; verify protected connection settings")
	}

	migrate(database)
	DB = database
	CreateOrUpdateAdminAccount()
}

func buildDatabaseDSN(user string, password string, host string, port string, name string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, name)
}

func migrate(database *gorm.DB) {
	if err := database.AutoMigrate(&models.LockerItem{}, &models.ClueComment{}, &models.ClueEdit{}, &models.LinkPreview{}, &models.CluePreviewReference{}); err != nil {
		panic(err)
	}
	if err := database.Migrator().AutoMigrate(&models.User{}); err != nil {
		panic(err)
	}
	migrateUserOptionalPassword(database)
	database.Migrator().AutoMigrate(&models.SampleModel{})
	database.Migrator().AutoMigrate(&models.SampleItem{})
	database.Migrator().AutoMigrate(&models.File{})
	database.Migrator().AutoMigrate(&models.Job{})
	database.Migrator().AutoMigrate(&models.FailedJob{})
	database.Migrator().AutoMigrate(&models.Category{})
	if err := database.AutoMigrate(&models.LivingClue{}); err != nil {
		panic(err)
	}
	migrateLivingClueOptionalLocation(database)
	if err := database.AutoMigrate(&models.ClueAlias{}, &models.ClueDiscoveryState{}, &models.ClueSuggestion{}); err != nil {
		panic(err)
	}
	if os.Getenv("ALLOW_DESTRUCTIVE_LEGACY_MIGRATIONS") == "true" {
		migrateRetiredGeography(database)
	}
	database.Migrator().AutoMigrate(&models.EnvironmentalContext{})
	database.Migrator().AutoMigrate(&models.ConnectionType{})
	database.Migrator().AutoMigrate(&models.ClueConnection{})
	database.Migrator().AutoMigrate(&models.NodeType{})
	database.Migrator().AutoMigrate(&models.Node{})
	database.Migrator().AutoMigrate(&models.NodeLink{})
	database.Migrator().AutoMigrate(&models.SensorReading{})
	if os.Getenv("ALLOW_DESTRUCTIVE_LEGACY_MIGRATIONS") == "true" {
		migrateNodeLinksHardDelete(database)
	}
	database.Migrator().AutoMigrate(&models.ChatTemplate{})
	database.Migrator().AutoMigrate(&models.ChatTemplateFile{})
	database.Migrator().AutoMigrate(&models.ChatMessage{})
	if err := database.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(&models.ParticipantConversation{}, &models.ParticipantMessage{}, &models.ParticipantConversationReport{}, &models.ParticipantConversationBlock{}); err != nil {
		log.Fatal(err)
	}
	database.Migrator().AutoMigrate(&models.AIMessage{})
	database.Migrator().AutoMigrate(&models.LivingClueChatMessage{})
}

func migrateUserOptionalPassword(database *gorm.DB) {
	columns, err := database.Migrator().ColumnTypes(&models.User{})
	if err != nil {
		panic(err)
	}
	for _, column := range columns {
		if column.Name() != "password" {
			continue
		}
		nullable, known := column.Nullable()
		if !known {
			panic("Cannot determine nullability of users.password")
		}
		if !nullable {
			if err := database.Migrator().AlterColumn(&models.User{}, "password"); err != nil {
				panic("Failed to allow optional users.password: " + err.Error())
			}
		}
		return
	}
	panic("users.password column not found")
}

// GORM's automatic migration does not relax existing NOT NULL columns in
// this version. Explicitly migrate legacy/restored databases, preserving values.
func migrateLivingClueOptionalLocation(database *gorm.DB) {
	columns, err := database.Migrator().ColumnTypes(&models.LivingClue{})
	if err != nil {
		panic(err)
	}
	for _, column := range columns {
		name := column.Name()
		if name != "latitude" && name != "longitude" {
			continue
		}
		nullable, known := column.Nullable()
		if !known {
			panic("Cannot determine nullability of living_clues." + name)
		}
		if !nullable {
			if err := database.Migrator().AlterColumn(&models.LivingClue{}, name); err != nil {
				panic("Failed to allow optional living_clues." + name + ": " + err.Error())
			}
		}
	}
}

// Remove manual geographic catalogs; Geo API context remains independent.
// MySQL DDL is not transactional: every step is checked and safe to rerun.
func migrateRetiredGeography(database *gorm.DB) {
	for _, column := range []string{"biome_id", "territory_id"} {
		if !database.Migrator().HasColumn(&models.LivingClue{}, column) {
			continue
		}
		var keys []string
		if err := database.Raw(`SELECT CONSTRAINT_NAME FROM information_schema.KEY_COLUMN_USAGE
   WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'living_clues'
   AND COLUMN_NAME = ? AND REFERENCED_TABLE_NAME IS NOT NULL`, column).Scan(&keys).Error; err != nil {
			panic(err)
		}
		for _, key := range keys {
			if !regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(key) {
				panic("Invalid legacy geography foreign key")
			}
			if err := database.Exec("ALTER TABLE living_clues DROP FOREIGN KEY `" + key + "`").Error; err != nil {
				panic(err)
			}
		}
		if err := database.Migrator().DropColumn(&models.LivingClue{}, column); err != nil {
			panic(err)
		}
	}
	for _, table := range []string{"territories", "biomes"} {
		if database.Migrator().HasTable(table) {
			if err := database.Migrator().DropTable(table); err != nil {
				panic(err)
			}
		}
	}
}

func migrateNodeLinksHardDelete(database *gorm.DB) {
	if !database.Migrator().HasTable(&models.NodeLink{}) {
		return
	}

	if database.Migrator().HasColumn(&models.NodeLink{}, "deleted_at") {
		_ = database.Exec("DELETE FROM node_links WHERE deleted_at IS NOT NULL").Error
		_ = database.Migrator().DropColumn(&models.NodeLink{}, "deleted_at")
	}
}

func CreateOrUpdateAdminAccount() {
	// Startup must not reset credentials, profile or user-maintained account data.
	var existing models.User
	err := DB.Unscoped().Where("email = ?", "admin@admin.com").First(&existing).Error
	if err == nil {
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		panic("Failed to inspect bootstrap account")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("uu2admin"), bcrypt.DefaultCost)
	if err != nil {
		panic("Failed to hahsed password!!! error:" + err.Error())
	}

	var user models.User
	DB.First(&user, "email = ?", "admin@admin.com")
	user.Name = "Admin"
	user.Email = "admin@admin.com"
	verifiedAt := time.Now()
	user.EmailVerifiedAt = &verifiedAt
	user.Password = string(hashedPassword)
	user.Profile = enums.UserProfileEnumAdministrator

	updateErr := DB.Save(&user).Error
	if updateErr != nil {
		panic("Failed to create admin user!!! error:" + updateErr.Error())
	}
}
