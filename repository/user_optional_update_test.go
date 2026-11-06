package repository

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUserUpdateClearsOnlyExplicitOptionalPhone(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: "tcp(127.0.0.1:1)/unused?parseTime=true", SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	previous := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previous })
	var statement string
	if err := db.Callback().Update().After("gorm:update").Register("capture_user_optional_update", func(tx *gorm.DB) { statement = tx.Statement.SQL.String() }); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, body   string
		phoneUpdated bool
	}{
		{"omitted", `{"name":"Edited","email":"synthetic@example.test","profile":10}`, false},
		{"null", `{"name":"Edited","email":"synthetic@example.test","profile":10,"phone":null}`, false},
		{"empty", `{"name":"Edited","email":"synthetic@example.test","profile":10,"phone":""}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var input requests.UpdateUserInput
			if err := json.Unmarshal([]byte(tc.body), &input); err != nil {
				t.Fatal(err)
			}
			user := &models.User{DefaultModel: models.DefaultModel{ID: 7}, Name: "Original", Phone: "00000000000"}
			if err := UpdateUser(user, &input); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(statement, "`phone`=?") != tc.phoneUpdated {
				t.Fatalf("Incorrect optional phone persistence for %s: %s", tc.name, statement)
			}
			for _, column := range []string{"`password`=", "`remember_token`=", "`fcm_token`=", "`reset_password_token`="} {
				if strings.Contains(statement, column) {
					t.Fatalf("Omitted credential field must be preserved: %s", statement)
				}
			}
		})
	}
}
