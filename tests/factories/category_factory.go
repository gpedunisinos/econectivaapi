package factories

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
)

func CreateCategory() (*models.Category, error) {
	category := &models.Category{
		Name:         fakers.Company(),
		Type:         fakers.Word(),
		ColorHex:     fakers.HexColor(),
		IconCode:     fakers.Word(),
		Description:  fakers.FullAddress(),
		MetadataTags: fakers.Word(),
	}

	err := config.DB.Create(&category).Error
	if err != nil {
		return nil, err
	}

	return category, nil
}
