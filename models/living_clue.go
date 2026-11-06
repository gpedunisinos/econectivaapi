package models

import "gorm.io/gorm"

type LivingClue struct {
	DefaultModel
	DeletedAt            gorm.DeletedAt        `gorm:"index" json:"deleted_at"`
	Name                 string                `gorm:"size:255;notNull" validate:"required" json:"name"`
	CategoryID           uint                  `gorm:"notNull" validate:"required" json:"category_id"`
	Category             *Category             `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"category"`
	Description          string                `gorm:"type:text" json:"description"`
	DescriptionDelta     string                `gorm:"type:longtext" json:"description_delta"`
	Latitude             *float64              `json:"latitude"`
	Longitude            *float64              `json:"longitude"`
	ImageURL             string                `gorm:"size:255" json:"image_url"`
	Metadata             JSONMap               `gorm:"type:json" json:"metadata"`
	IsActive             bool                  `gorm:"default:true" json:"is_active"`
	Files                []File                `json:"files"`
	SensorReadings       []SensorReading       `gorm:"-" json:"sensor_readings,omitempty"`
	EnvironmentalContext *EnvironmentalContext `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"environmental_context,omitempty"`
}
