package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"gorm.io/gorm"
)

func FindAIMessagesByUserID(userID uint, limit int, offset int) ([]models.AIMessage, error) {
	var messages []models.AIMessage
	query := config.DB.Where("user_id = ?", userID).Order("id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	err := query.Find(&messages).Error
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func CreateAIMessage(tx *gorm.DB, message *models.AIMessage) error {
	return tx.Create(message).Error
}
