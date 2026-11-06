package factories

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
)

func CreateFile() (*models.File, error) {
	model := &models.File{
		Extension: "jpg",
		Path:      "uploads/users/1/profile_picture.jpg",
		PublicURL: "http://localhost:8080/storage/uploads/users/1/profile_picture.jpg",
		Name:      fakers.RandomString(),
	}
	err := config.DB.Create(&model).Error
	if err != nil {
		return nil, err
	}
	return model, nil
}
