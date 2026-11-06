package factories

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
)

func CreateNodeType() (*models.NodeType, error) {
	nodeType := &models.NodeType{
		Name:  fakers.Word(),
		Color: fakers.HexColor(),
	}

	err := config.DB.Create(&nodeType).Error
	if err != nil {
		return nil, err
	}

	return nodeType, nil
}
