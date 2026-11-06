package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"gorm.io/gorm"
)

func FindChatMessagesByTemplateAndUserID(chatTemplateID uint, userID uint, limit int, offset int) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	query := config.DB.Where("chat_template_id = ? AND user_id = ?", chatTemplateID, userID).Order("id desc")
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
	// Retorna em ordem cronológica ascendente para o app renderizar naturalmente
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func CreateChatMessage(tx *gorm.DB, message *models.ChatMessage) error {
	return tx.Create(message).Error
}

func DeleteChatMessagesByTemplateAndUserID(chatTemplateID uint, userID uint) error {
	return config.DB.Where("chat_template_id = ? AND user_id = ?", chatTemplateID, userID).Delete(&models.ChatMessage{}).Error
}
