package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"strings"
)

func FindChatTemplates(limit int, offset int, search ...string) ([]models.ChatTemplate, error) {
	var templates []models.ChatTemplate
	query := config.DB.Preload("ChatTemplateFiles").Order("id desc")
	if len(search) > 0 {
		if term := strings.TrimSpace(search[0]); term != "" {
			// Treat LIKE wildcard characters literally; user input remains parameterized.
			term = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(term)
			pattern := "%" + term + "%"
			query = query.Where("(name LIKE ? ESCAPE '!' OR description LIKE ? ESCAPE '!')", pattern, pattern)
		}
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	err := query.Find(&templates).Error
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func FindChatTemplateByID(id string) (*models.ChatTemplate, error) {
	var template models.ChatTemplate
	err := config.DB.Preload("ChatTemplateFiles").Where("id = ?", id).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func FindChatTemplateByIDAndUserID(id string, userID uint) (*models.ChatTemplate, error) {
	var template models.ChatTemplate
	err := config.DB.Preload("ChatTemplateFiles").Where("id = ? AND user_id = ?", id, userID).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func CreateChatTemplate(template *models.ChatTemplate) error {
	return config.DB.Create(template).Error
}

func DeleteChatTemplate(template *models.ChatTemplate) error {
	return config.DB.Delete(template).Error
}
