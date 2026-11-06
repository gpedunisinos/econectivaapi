package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SaveEnvironmentalContext(context *models.EnvironmentalContext) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		var clue models.LivingClue
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&clue, context.LivingClueID).Error; err != nil {
			return err
		}
		if clue.Latitude == nil || clue.Longitude == nil || *clue.Latitude != context.AnalyzedLatitude || *clue.Longitude != context.AnalyzedLongitude {
			return nil
		}
		result := tx.Model(&models.EnvironmentalContext{}).Where("id = ?", context.ID).Select("*").Updates(context)
		return result.Error
	})
}

func ResetEnvironmentalContext(snapshot *models.LivingClue) (*models.EnvironmentalContext, error) {
	var context *models.EnvironmentalContext
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var clue models.LivingClue
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&clue, snapshot.ID).Error; err != nil {
			return err
		}
		if clue.Latitude == nil || clue.Longitude == nil || snapshot.Latitude == nil || snapshot.Longitude == nil || *clue.Latitude != *snapshot.Latitude || *clue.Longitude != *snapshot.Longitude {
			return nil
		}
		// A new row is a generation token: old workers cannot write after reprocess,
		// including an A -> B -> A move back to identical coordinates.
		if err := tx.Where("living_clue_id = ?", clue.ID).Delete(&models.EnvironmentalContext{}).Error; err != nil {
			return err
		}
		context = &models.EnvironmentalContext{LivingClueID: clue.ID, Status: models.EnvironmentalContextStatusPending, AnalyzedLatitude: *clue.Latitude, AnalyzedLongitude: *clue.Longitude, Sources: models.JSONMap{}}
		return tx.Create(context).Error
	})
	return context, err
}

func FindEnvironmentalContextByLivingClueID(livingClueID uint) (*models.EnvironmentalContext, error) {
	var context models.EnvironmentalContext
	err := config.DB.Where("living_clue_id = ?", livingClueID).First(&context).Error
	return &context, err
}
