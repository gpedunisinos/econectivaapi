package factories

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
)

func CreateConnectionType() (*models.ConnectionType, error) {
	connectionType := &models.ConnectionType{
		Name:          fakers.UUID(),
		StrokeStyle:   "solid",
		IsDirectional: fakers.Bool(),
	}

	err := config.DB.Create(&connectionType).Error
	if err != nil {
		return nil, err
	}

	return connectionType, nil
}
