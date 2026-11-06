package models

import "gorm.io/gorm"

type Category struct {
	DefaultModel
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Name         string         `gorm:"size:255;notNull" validate:"required" json:"name"`
	Type         string         `gorm:"size:100;notNull;default:''" validate:"required" json:"type"`
	ColorHex     string         `gorm:"size:7;notNull" validate:"required" json:"color_hex"`
	IconCode     string         `gorm:"size:255" json:"icon_code"`
	Description  string         `gorm:"type:text" json:"description"`
	MetadataTags string         `gorm:"type:text;notNull" validate:"required" json:"metadata_tags"`
}
