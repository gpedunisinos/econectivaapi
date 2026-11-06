package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"gorm.io/gorm"
)

func FindClueConnections(search string, limit *int, offset *int) ([]models.ClueConnection, error) {
	var clueConnections []models.ClueConnection
	query := config.DB.Preload("SourceClue").Preload("TargetClue").Preload("ConnectionType").Order("id desc")
	if search != "" {
		query = query.Where("clue_connections.id LIKE ? OR clue_connections.evidence LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}
	err := query.Find(&clueConnections).Error
	if err != nil {
		return nil, err
	}
	return clueConnections, nil
}

func FindClueConnectionsByClueID(clueID uint) ([]models.ClueConnection, error) {
	var clueConnections []models.ClueConnection
	err := config.DB.
		Preload("SourceClue.Category").
		Preload("TargetClue.Category").
		Preload("ConnectionType").
		Where("source_clue_id = ? OR target_clue_id = ?", clueID, clueID).
		Order("id desc").
		Find(&clueConnections).Error
	if err != nil {
		return nil, err
	}
	return clueConnections, nil
}

func FindClueConnectionByID(ID string) (*models.ClueConnection, error) {
	var clueConnection models.ClueConnection
	err := config.DB.Preload("SourceClue").Preload("TargetClue").Preload("ConnectionType").Where("clue_connections.id = ?", ID).First(&clueConnection).Error
	if err != nil {
		return nil, err
	}
	return &clueConnection, nil
}

func CreateClueConnection(clueConnection *models.ClueConnection) error {
	if clueConnection.Strength != 0 {
		return config.DB.Create(&clueConnection).Error
	}
	// GORM substitutes the model default for zero during creation. Keep the
	// controller's explicit zero atomic with the insert; omitted input is1.
	return config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(clueConnection).Error; err != nil {
			return err
		}
		if err := tx.Model(clueConnection).UpdateColumn("strength", 0).Error; err != nil {
			return err
		}
		clueConnection.Strength = 0
		return nil
	})
}

func UpdateClueConnection(clueConnection *models.ClueConnection, updatedClueConnection *models.ClueConnection) error {
	return config.DB.Model(clueConnection).Select("source_clue_id", "target_clue_id", "connection_type_id", "strength", "evidence").Updates(updatedClueConnection).Error
}

func DeleteClueConnection(clueConnection *models.ClueConnection) error {
	return config.DB.Delete(&clueConnection).Error
}

func ClueConnectionExists(sourceClueID uint, targetClueID uint, connectionTypeID uint, ignoreID uint) (bool, error) {
	var count int64
	query := config.DB.Model(&models.ClueConnection{}).
		Where("source_clue_id = ? AND target_clue_id = ? AND connection_type_id = ?", sourceClueID, targetClueID, connectionTypeID)
	if ignoreID > 0 {
		query = query.Where("id <> ?", ignoreID)
	}
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
