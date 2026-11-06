package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func FindChatTemplateFilesByChatTemplateID(chatTemplateID uint) ([]models.ChatTemplateFile, error) {
	var files []models.ChatTemplateFile
	err := config.DB.Where("chat_template_id = ?", chatTemplateID).Order("id desc").Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}

func FindChatTemplateFileByIDAndChatTemplateID(id uint, chatTemplateID uint) (*models.ChatTemplateFile, error) {
	var file models.ChatTemplateFile
	err := config.DB.Where("id = ? AND chat_template_id = ?", id, chatTemplateID).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func FindChatTemplateFileByID(id uint) (*models.ChatTemplateFile, error) {
	var file models.ChatTemplateFile
	err := config.DB.First(&file, id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func CreateChatTemplateFile(file *models.ChatTemplateFile) error {
	return config.DB.Create(file).Error
}

func DeleteChatTemplateFile(file *models.ChatTemplateFile) error {
	return config.DB.Delete(file).Error
}
