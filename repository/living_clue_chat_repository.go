package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"gorm.io/gorm"
)

func FindLivingClueChatMessagesByLivingClueAndUserID(livingClueID uint, userID uint, limit int, offset int) ([]models.LivingClueChatMessage, error) {
	var messages []models.LivingClueChatMessage
	query := config.DB.Where("living_clue_id = ? AND user_id = ?", livingClueID, userID).Order("id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func CreateLivingClueChatMessage(tx *gorm.DB, message *models.LivingClueChatMessage) error {
	return tx.Create(message).Error
}

func DeleteLivingClueChatMessagesByLivingClueAndUserID(livingClueID uint, userID uint) error {
	return config.DB.Where("living_clue_id = ? AND user_id = ?", livingClueID, userID).Delete(&models.LivingClueChatMessage{}).Error
}
