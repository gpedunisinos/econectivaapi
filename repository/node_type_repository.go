package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func FindNodeTypes(search string, limit *int, offset *int) ([]models.NodeType, error) {
	var nodeTypes []models.NodeType
	query := config.DB.Order("name")
	if search != "" {
		query = query.Where("name LIKE ? OR id LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}
	err := query.Find(&nodeTypes).Error
	if err != nil {
		return nil, err
	}
	return nodeTypes, nil
}

func FindNodeTypeByID(ID string) (*models.NodeType, error) {
	var nodeType models.NodeType
	err := config.DB.Where("id = ?", ID).First(&nodeType).Error
	if err != nil {
		return nil, err
	}
	return &nodeType, nil
}

func FindNodeTypeByName(name string) (*models.NodeType, error) {
	var nodeType models.NodeType
	err := config.DB.Where("name = ?", name).First(&nodeType).Error
	if err != nil {
		return nil, err
	}
	return &nodeType, nil
}

func CreateNodeType(nodeType *models.NodeType) error {
	return config.DB.Create(&nodeType).Error
}

func UpdateNodeType(nodeType *models.NodeType, updatedNodeType *models.NodeType) error {
	return config.DB.Model(&nodeType).Updates(updatedNodeType).Error
}

func DeleteNodeType(nodeType *models.NodeType) error {
	return config.DB.Delete(&nodeType).Error
}
