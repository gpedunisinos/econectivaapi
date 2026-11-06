// tests/factories/sample_factory.go

package factories

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
)

func CreateSampleDetail() (*models.SampleDetail, error) {
	model := &models.SampleDetail{
		SampleString: fakers.Word(),
	}
	err := config.DB.Create(&model).Error
	if err != nil {
		return nil, err
	}

	return model, nil
}
