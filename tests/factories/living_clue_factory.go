package factories

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
)

func CreateLivingClue() (*models.LivingClue, error) {
	category, err := CreateCategory()
	if err != nil {
		return nil, err
	}
	livingClue := &models.LivingClue{Name: fakers.Name(), CategoryID: category.ID, Description: fakers.FullAddress(), Latitude: floatPtr(fakers.Latitude()), Longitude: floatPtr(fakers.Longitude()), ImageURL: fakers.URL(), Metadata: models.JSONMap{"info": fakers.Word()}, IsActive: true}
	if err := config.DB.Create(&livingClue).Error; err != nil {
		return nil, err
	}
	return livingClue, nil
}

func floatPtr(value float64) *float64 { return &value }
