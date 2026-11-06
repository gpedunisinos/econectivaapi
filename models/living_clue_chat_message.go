package models

import "gorm.io/gorm"

type LivingClueChatMessage struct {
	DefaultModel
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	LivingClueID uint           `gorm:"not null;index" json:"living_clue_id"`
	UserID       uint           `gorm:"not null;index" json:"user_id"`
	Role         string         `gorm:"size:20;not null" binding:"required" json:"role"`
	Content      string         `gorm:"type:longtext;not null" binding:"required" json:"content"`
}
