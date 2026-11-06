package tests

import (
	"encoding/json"
	"fmt"
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClueHistoryCreationAndLegacyPagination(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()
	category, err := factories.CreateCategory()
	require.NoError(t, err)
	clue := models.LivingClue{Name: "Pista com histórico", CategoryID: category.ID, IsActive: true}
	require.NoError(t, repository.CreateLivingClue(&clue, user.ID))
	fetch := func(id uint, suffix string) []models.ClueEdit {
		t.Helper()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/living_clues/%d/edits%s", id, suffix), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, 200, w.Code, w.Body.String())
		var events []models.ClueEdit
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &events))
		return events
	}
	events := fetch(clue.ID, "")
	require.Len(t, events, 1)
	require.Equal(t, "created", events[0].EventType)
	require.Equal(t, "Pista criada", events[0].Description)
	require.Equal(t, user.ID, events[0].Author.ID)
	require.WithinDuration(t, clue.CreatedAt, events[0].CreatedAt, time.Second)
	updated := clue
	updated.Name = "Pista revisada"
	require.NoError(t, repository.UpdateLivingClueWithEdit(&clue, &updated, user.ID, false))
	events = fetch(clue.ID, "")
	require.Len(t, events, 2)
	require.Equal(t, "updated", events[0].EventType)
	require.Equal(t, "created", events[1].EventType)
	// Legacy records expose their known timestamp, never a fabricated author.
	legacy := models.LivingClue{Name: "Legado", CategoryID: category.ID, IsActive: true, DefaultModel: models.DefaultModel{CreatedAt: time.Now().Add(-24 * time.Hour)}}
	require.NoError(t, config.DB.Create(&legacy).Error)
	for i := 0; i < 2; i++ {
		require.NoError(t, config.DB.Create(&models.ClueEdit{LivingClueID: legacy.ID, UserID: user.ID}).Error)
	}
	require.Len(t, fetch(legacy.ID, "?limit=2&offset=0"), 2)
	events = fetch(legacy.ID, "?limit=2&offset=2")
	require.Len(t, events, 1)
	require.Equal(t, "created", events[0].EventType)
	require.Zero(t, events[0].ID)
	require.Zero(t, events[0].Author.ID)
	require.Equal(t, "Autoria não registrada", events[0].Author.Name)
	require.WithinDuration(t, legacy.CreatedAt, events[0].CreatedAt, time.Second)
	require.Empty(t, fetch(legacy.ID, "?limit=2&offset=3"))
	var count int64
	require.NoError(t, config.DB.Model(&models.ClueEdit{}).Where("living_clue_id = ?", legacy.ID).Count(&count).Error)
	require.EqualValues(t, 2, count)
	empty := models.LivingClue{Name: "Legado sem edição", CategoryID: category.ID, IsActive: true}
	require.NoError(t, config.DB.Create(&empty).Error)
	require.Len(t, fetch(empty.ID, ""), 1)
}

func TestClueHistoryCreationIsAtomic(t *testing.T) {
	setupDatabase()
	category, err := factories.CreateCategory()
	require.NoError(t, err)
	callback := "test:fail_creation_history"
	require.NoError(t, config.DB.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "clue_edits" {
			tx.AddError(fmt.Errorf("synthetic history failure"))
		}
	}))
	defer config.DB.Callback().Create().Remove(callback)
	clue := models.LivingClue{Name: "Criação deve reverter", CategoryID: category.ID}
	require.Error(t, repository.CreateLivingClue(&clue, 123))
	var count int64
	require.NoError(t, config.DB.Model(&models.LivingClue{}).Where("id = ?", clue.ID).Count(&count).Error)
	require.Zero(t, count)
}
