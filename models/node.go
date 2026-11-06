package models

import "gorm.io/gorm"

type Node struct {
	DefaultModel
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Name       string         `gorm:"size:255;notNull" validate:"required" json:"name"`
	Color      string         `gorm:"size:20;notNull" validate:"required" json:"color"`
	PositionX  *float64       `gorm:"type:double" json:"x"`
	PositionY  *float64       `gorm:"type:double" json:"y"`
	NodeTypeID uint           `gorm:"notNull" validate:"required" json:"node_type_id"`
	NodeType   *NodeType      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"node_type"`
}
