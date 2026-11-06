package factories

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func CreateNodeLink() (*models.NodeLink, error) {
	sourceNode, err := CreateNode()
	if err != nil {
		return nil, err
	}

	targetNode, err := CreateNode()
	if err != nil {
		return nil, err
	}

	nodeLink := &models.NodeLink{
		SourceNodeID: sourceNode.ID,
		TargetNodeID: targetNode.ID,
	}

	err = config.DB.Create(&nodeLink).Error
	if err != nil {
		return nil, err
	}

	return nodeLink, nil
}
