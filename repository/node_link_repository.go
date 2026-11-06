package repository

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

func FindNodeLinks(search string, limit *int, offset *int) ([]models.NodeLink, error) {
	var nodeLinks []models.NodeLink
	query := config.DB.
		Preload("SourceNode").
		Preload("SourceNode.NodeType").
		Preload("TargetNode").
		Preload("TargetNode.NodeType").
		Order("id desc")
	if search != "" {
		query = query.Where("node_links.id LIKE ?", "%"+search+"%")
	}
	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}
	if offset != nil && *offset > 0 {
		query = query.Offset(*offset)
	}
	err := query.Find(&nodeLinks).Error
	if err != nil {
		return nil, err
	}
	return nodeLinks, nil
}

func FindNodeLinkByID(ID string) (*models.NodeLink, error) {
	var nodeLink models.NodeLink
	err := config.DB.
		Preload("SourceNode").
		Preload("SourceNode.NodeType").
		Preload("TargetNode").
		Preload("TargetNode.NodeType").
		Where("node_links.id = ?", ID).
		First(&nodeLink).Error
	if err != nil {
		return nil, err
	}
	return &nodeLink, nil
}

func CreateNodeLink(nodeLink *models.NodeLink) error {
	return config.DB.Create(&nodeLink).Error
}

func UpdateNodeLink(nodeLink *models.NodeLink, updatedNodeLink *models.NodeLink) error {
	return config.DB.Model(&nodeLink).Updates(updatedNodeLink).Error
}

func DeleteNodeLink(nodeLink *models.NodeLink) error {
	return config.DB.Delete(&nodeLink).Error
}

func NodeLinkExists(sourceNodeID uint, targetNodeID uint, ignoreID uint) (bool, error) {
	var count int64
	query := config.DB.Model(&models.NodeLink{}).Where(
		"(source_node_id = ? AND target_node_id = ?) OR (source_node_id = ? AND target_node_id = ?)",
		sourceNodeID, targetNodeID, targetNodeID, sourceNodeID,
	)
	if ignoreID > 0 {
		query = query.Where("id <> ?", ignoreID)
	}
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func FindNodesMapData() (nodes []models.Node, nodeLinks []models.NodeLink, err error) {
	err = config.DB.Preload("NodeType").Order("name").Find(&nodes).Error
	if err != nil {
		return nil, nil, err
	}

	err = config.DB.
		Preload("SourceNode").
		Preload("SourceNode.NodeType").
		Preload("TargetNode").
		Preload("TargetNode.NodeType").
		Order("id desc").
		Find(&nodeLinks).Error
	if err != nil {
		return nil, nil, err
	}

	return nodes, nodeLinks, nil
}
