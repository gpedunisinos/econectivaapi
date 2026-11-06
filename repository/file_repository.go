package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func FindFiles() (*[]models.File, error) {
	var files []models.File
	err := config.DB.Preload("LivingClue").Find(&files).Error
	if err != nil {
		return nil, err
	}
	return &files, nil
}

func CreateFile(file *models.File) error {
	return config.DB.Create(file).Error
}

func FindFileByID(id uint) (*models.File, error) {
	var file models.File
	if err := config.DB.First(&file, id).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func DeleteFile(file *models.File) error {
	return config.DB.Delete(file).Error
}
