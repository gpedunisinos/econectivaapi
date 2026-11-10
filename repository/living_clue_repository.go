package repository

import (
	"crypto/sha256"
	"fmt"
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func applyLivingClueFilters(query *gorm.DB, search string, categoryID *uint) *gorm.DB {
	query = query.Preload("Category").Preload("EnvironmentalContext")
	if search != "" {
		query = query.Where("living_clues.name LIKE ? OR living_clues.description LIKE ? OR living_clues.id LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if categoryID != nil && *categoryID > 0 {
		query = query.Where("living_clues.category_id = ?", *categoryID)
	}
	return query
}

func FindLivingClues(search string, categoryID *uint, limit *int, offset *int) ([]models.LivingClue, error) {
	var livingClues []models.LivingClue
	query := applyLivingClueFilters(config.DB.Preload("Files").Preload("Category").Order("name"), search, categoryID)
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}
	if err := query.Find(&livingClues).Error; err != nil {
		return nil, err
	}
	return livingClues, nil
}

func FindLivingCluesByBounds(search string, categoryID *uint, north float64, south float64, east float64, west float64, limit *int, offset *int) ([]models.LivingClue, error) {
	var livingClues []models.LivingClue
	query := applyLivingClueFilters(config.DB.Preload("Files").Preload("Category").Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?", south, north, west, east).Order("name"), search, categoryID)
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}
	if err := query.Find(&livingClues).Error; err != nil {
		return nil, err
	}
	return livingClues, nil
}

func FindLivingCluesByRadius(search string, categoryID *uint, latitude float64, longitude float64, radiusMeters float64, limit *int, offset *int) ([]models.LivingClue, error) {
	var livingClues []models.LivingClue
	query := applyLivingClueFilters(config.DB.Preload("Files").Preload("Category").Where("(6371000 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) <= ?", latitude, longitude, latitude, radiusMeters).Order("name"), search, categoryID)
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}
	if err := query.Find(&livingClues).Error; err != nil {
		return nil, err
	}
	return livingClues, nil
}

func FindLivingClueByID(ID string) (*models.LivingClue, error) {
	var livingClue models.LivingClue
	err := config.DB.Preload("Aliases", "status = ?", "approved").Preload("Category").Preload("Files").Preload("EnvironmentalContext").Where("living_clues.id = ?", ID).First(&livingClue).Error
	if err != nil {
		return nil, err
	}
	sensorReadings, err := FindLastSensorReadingsByLivingClueID(livingClue.ID, 10)
	if err == nil {
		livingClue.SensorReadings = sensorReadings
	}
	return &livingClue, nil
}

func FindLivingClueReferenceByID(ID string) (*models.LivingClue, error) {
	var livingClue models.LivingClue
	if err := config.DB.Where("living_clues.id = ?", ID).First(&livingClue).Error; err != nil {
		return nil, err
	}
	return &livingClue, nil
}

func CreateLivingClue(livingClue *models.LivingClue, creatorIDs ...uint) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(livingClue).Error; err != nil {
			return err
		}
		if err := syncCluePreviewReferences(tx, livingClue.ID, livingClue); err != nil {
			return err
		}
		var creatorID uint
		if len(creatorIDs) > 0 {
			creatorID = creatorIDs[0]
		}
		if err := services.ReplaceClueAliases(tx, livingClue); err != nil {
			return err
		}
		return tx.Create(&models.ClueEdit{LivingClueID: livingClue.ID, UserID: creatorID, EventType: "created", CreatedAt: livingClue.CreatedAt}).Error
	})
}
func UpdateLivingClue(livingClue *models.LivingClue, updatedLivingClue *models.LivingClue) error {
	return config.DB.Model(&livingClue).
		Select(
			"name",
			"category_id",
			"description",
			"description_delta",
			"latitude",
			"longitude",
			"image_url",
			"metadata",
			"is_active",
		).
		Updates(updatedLivingClue).Error
}
func DeleteLivingClue(livingClue *models.LivingClue) error {
	return config.DB.Delete(&livingClue).Error
}

func FindLivingClueSelects(search string, categoryID *uint, limit *int, offset *int) ([]models.ItemSelect, error) {
	livingClues, err := FindLivingClues(search, categoryID, limit, offset)
	if err != nil {
		return nil, err
	}
	var itemSelects []models.ItemSelect
	for _, livingClue := range livingClues {
		itemSelects = append(itemSelects, models.ItemSelect{ID: livingClue.ID, Name: livingClue.Name})
	}
	return itemSelects, nil
}

// Edit and event commit atomically. Invalidation shares the clue lock with background results.
func UpdateLivingClueWithEdit(clue, updated *models.LivingClue, userID uint, locationChanged bool) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.LivingClue
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, clue.ID).Error; err != nil {
			return err
		}
		if err := tx.Model(&locked).Select("name", "category_id", "description", "description_delta", "latitude", "longitude", "image_url", "metadata", "is_active").Updates(updated).Error; err != nil {
			return err
		}
		if locationChanged {
			if err := tx.Where("living_clue_id = ?", clue.ID).Delete(&models.EnvironmentalContext{}).Error; err != nil {
				return err
			}
		}
		if err := syncCluePreviewReferences(tx, clue.ID, updated); err != nil {
			return err
		}
		updated.ID = clue.ID
		if err := services.ReplaceClueAliases(tx, updated); err != nil {
			return err
		}
		return tx.Create(&models.ClueEdit{LivingClueID: clue.ID, UserID: userID, EventType: "updated"}).Error
	})
}

func syncCluePreviewReferences(tx *gorm.DB, id uint, clue *models.LivingClue) error {
	if err := tx.Where("living_clue_id = ?", id).Delete(&models.CluePreviewReference{}).Error; err != nil {
		return err
	}
	for _, url := range services.QuillPreviewURLs(clue.DescriptionDelta, clue.Description) {
		reference := models.CluePreviewReference{LivingClueID: id, URL: url, URLHash: fmt.Sprintf("%x", sha256.Sum256([]byte(url)))}
		if err := tx.Create(&reference).Error; err != nil {
			return err
		}
	}
	return nil
}

// SyncImportedCluePreviews updates only the local preview-reference index.
func SyncImportedCluePreviews(tx *gorm.DB, clue *models.LivingClue) error {
	return syncCluePreviewReferences(tx, clue.ID, clue)
}
