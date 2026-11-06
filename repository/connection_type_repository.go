package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"gorm.io/gorm"
)

func FindConnectionTypes(search string, limit *int, offset *int) ([]models.ConnectionType, error) {
	var connectionTypes []models.ConnectionType
	query := config.DB.Order("name")
	if search != "" {
		query = query.Where("name LIKE ? OR id LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}
	err := query.Find(&connectionTypes).Error
	if err != nil {
		return nil, err
	}
	return connectionTypes, nil
}

func FindConnectionTypeByID(ID string) (*models.ConnectionType, error) {
	var connectionType models.ConnectionType
	err := config.DB.Where("id = ?", ID).First(&connectionType).Error
	if err != nil {
		return nil, err
	}
	return &connectionType, nil
}

func FindConnectionTypeByName(name string) (*models.ConnectionType, error) {
	var connectionType models.ConnectionType
	err := config.DB.Where("name = ?", name).First(&connectionType).Error
	if err != nil {
		return nil, err
	}
	return &connectionType, nil
}

func CreateConnectionType(connectionType *models.ConnectionType) error {
	// GORM substitutes the schema's true default for a false bool on insert.
	// Retain the validated caller choice atomically without changing DB defaults.
	directional := connectionType.IsDirectional
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(connectionType).Error; err != nil {
			return err
		}
		if !directional {
			return tx.Model(connectionType).UpdateColumn("is_directional", false).Error
		}
		return nil
	})
	connectionType.IsDirectional = directional
	return err
}

func UpdateConnectionType(connectionType *models.ConnectionType, updatedConnectionType *models.ConnectionType) error {
	return config.DB.Model(connectionType).Select("name", "stroke_style", "is_directional").Updates(updatedConnectionType).Error
}

func DeleteConnectionType(connectionType *models.ConnectionType) error {
	return config.DB.Delete(&connectionType).Error
}

func FindConnectionTypeSelects(search string, limit *int, offset *int) ([]models.ItemSelect, error) {
	connectionTypes, err := FindConnectionTypes(search, limit, offset)
	if err != nil {
		return nil, err
	}
	var itemSelects []models.ItemSelect
	for _, connectionType := range connectionTypes {
		itemSelects = append(itemSelects, models.ItemSelect{
			ID:   connectionType.ID,
			Name: connectionType.Name,
		})
	}
	return itemSelects, nil
}
