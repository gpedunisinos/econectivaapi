package factories

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/fakers"
)

func CreateNode() (*models.Node, error) {
	nodeType, err := CreateNodeType()
	if err != nil {
		return nil, err
	}

	node := &models.Node{
		Name:       fakers.Name(),
		Color:      fakers.HexColor(),
		NodeTypeID: nodeType.ID,
	}

	err = config.DB.Create(&node).Error
	if err != nil {
		return nil, err
	}

	return node, nil
}
