package models

import "gorm.io/gorm"

type ClueConnection struct {
	DefaultModel
	DeletedAt        gorm.DeletedAt  `gorm:"index" json:"deleted_at"`
	SourceClueID     uint            `gorm:"notNull;uniqueIndex:idx_clue_connection_triplet" json:"source_clue_id"`
	TargetClueID     uint            `gorm:"notNull;uniqueIndex:idx_clue_connection_triplet" json:"target_clue_id"`
	ConnectionTypeID uint            `gorm:"notNull;uniqueIndex:idx_clue_connection_triplet" json:"connection_type_id"`
	Strength         int             `gorm:"default:1" json:"strength"`
	Evidence         string          `gorm:"type:text" json:"evidence"`
	SourceClue       *LivingClue     `gorm:"foreignKey:SourceClueID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"source_clue"`
	TargetClue       *LivingClue     `gorm:"foreignKey:TargetClueID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"target_clue"`
	ConnectionType   *ConnectionType `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"connection_type"`
}
