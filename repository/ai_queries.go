package repository

import (
	"context"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
)

type ClueBounds struct{ North, South, East, West float64 }

// AI queries reuse the normal repository filters, with SQL-level bounds and
// request cancellation. They never select a user's messages or personal locker.
func FindAIClues(ctx context.Context, search string, category *uint, bounds *ClueBounds, limit, offset int) ([]models.LivingClue, error) {
	items := []models.LivingClue{}
	query := applyLivingClueFilters(config.DB.WithContext(ctx), search, category).Order("living_clues.name, living_clues.id").Limit(limit).Offset(offset)
	if bounds != nil {
		query = query.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?", bounds.South, bounds.North, bounds.West, bounds.East)
	}
	err := query.Find(&items).Error
	return items, err
}

func FindAIClue(ctx context.Context, id uint) (*models.LivingClue, error) {
	clue := &models.LivingClue{}
	err := config.DB.WithContext(ctx).Preload("Category").Preload("Files").Preload("EnvironmentalContext").First(clue, id).Error
	if err != nil {
		return nil, err
	}
	_ = config.DB.WithContext(ctx).Where("living_clue_id = ?", id).Order("id DESC").Limit(5).Find(&clue.SensorReadings).Error
	return clue, nil
}

func FindAIClueConnections(ctx context.Context, id uint, limit, offset int) ([]models.ClueConnection, error) {
	items := []models.ClueConnection{}
	err := config.DB.WithContext(ctx).Preload("SourceClue").Preload("TargetClue").Preload("ConnectionType").Where("source_clue_id = ? OR target_clue_id = ?", id, id).Order("id DESC").Limit(limit).Offset(offset).Find(&items).Error
	return items, err
}

func FindAINodes(ctx context.Context, search string, nodeType *uint, limit, offset int) ([]models.Node, error) {
	items := []models.Node{}
	query := config.DB.WithContext(ctx).Preload("NodeType").Order("nodes.name, nodes.id").Limit(limit).Offset(offset)
	if search != "" {
		query = query.Where("nodes.name LIKE ? OR nodes.id LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if nodeType != nil {
		query = query.Where("node_type_id = ?", *nodeType)
	}
	err := query.Find(&items).Error
	return items, err
}

func FindAINode(ctx context.Context, id uint) (*models.Node, []models.NodeLink, error) {
	node := &models.Node{}
	if err := config.DB.WithContext(ctx).Preload("NodeType").First(node, id).Error; err != nil {
		return nil, nil, err
	}
	links := []models.NodeLink{}
	err := config.DB.WithContext(ctx).Preload("SourceNode.NodeType").Preload("TargetNode.NodeType").Where("source_node_id = ? OR target_node_id = ?", id, id).Order("id DESC").Limit(20).Find(&links).Error
	return node, links, err
}
