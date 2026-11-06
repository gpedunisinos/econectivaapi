package models

import "gorm.io/gorm"

type NodeType struct {
	DefaultModel
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Name      string         `gorm:"size:100;notNull" validate:"required" json:"name"`
	Color     string         `gorm:"size:20;notNull" validate:"required" json:"color"`
}
