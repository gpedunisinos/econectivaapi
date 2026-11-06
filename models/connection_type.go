package models

import "gorm.io/gorm"

type ConnectionType struct {
	DefaultModel
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Name          string         `gorm:"size:100;notNull" validate:"required" json:"name"`
	StrokeStyle   string         `gorm:"size:50;default:solid" json:"stroke_style"`
	IsDirectional bool           `gorm:"default:true" json:"is_directional"`
}
