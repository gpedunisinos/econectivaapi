package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func FindNodes(search string, limit *int, offset *int) ([]models.Node, error) {
	var nodes []models.Node
	query := config.DB.Preload("NodeType").Order("name")
	if search != "" {
		query = query.Where("nodes.name LIKE ? OR nodes.id LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}
	err := query.Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func FindNodeByID(ID string) (*models.Node, error) {
	var node models.Node
	err := config.DB.Preload("NodeType").Where("nodes.id = ?", ID).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func CreateNode(node *models.Node) error {
	return config.DB.Create(&node).Error
}

func UpdateNode(node *models.Node, updatedNode *models.Node) error {
	return config.DB.Model(&node).Updates(updatedNode).Error
}

func DeleteNode(node *models.Node) error {
	return config.DB.Delete(&node).Error
}
