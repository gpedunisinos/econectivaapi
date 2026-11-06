package models

import "gorm.io/gorm"

type SensorReading struct {
	DefaultModel
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	LivingClueID uint           `gorm:"notNull;index" json:"living_clue_id"`
	ContentType  string         `gorm:"size:255" json:"content_type"`
	PayloadRaw   string         `gorm:"type:longtext;notNull" json:"payload_raw"`
}
