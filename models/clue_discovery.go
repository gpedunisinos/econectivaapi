package models

import "time"

type ClueAlias struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	LivingClueID   uint   `gorm:"uniqueIndex:idx_clue_alias_name;not null" json:"living_clue_id"`
	Name           string `gorm:"size:120;not null" json:"name"`
	NormalizedName string `gorm:"size:160;uniqueIndex:idx_clue_alias_name;not null" json:"-"`
	Status         string `gorm:"size:16;not null" json:"status"`
	Origin         string `gorm:"size:24;not null" json:"origin"`
	Evidence       string `gorm:"type:text" json:"evidence,omitempty"`
}

type ClueDiscoveryState struct {
	LivingClueID uint      `gorm:"primaryKey" json:"-"`
	ContentHash  string    `gorm:"size:64" json:"-"`
	Status       string    `gorm:"size:16" json:"status"`
	Unresolved   JSONMap   `gorm:"type:json" json:"-"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ClueSuggestion struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	SourceClueID uint        `gorm:"uniqueIndex:idx_clue_suggestion_pair" json:"-"`
	TargetClueID uint        `gorm:"uniqueIndex:idx_clue_suggestion_pair" json:"target_clue_id"`
	SourceHash   string      `gorm:"size:64" json:"-"`
	Score        int         `json:"-"`
	Kind         string      `gorm:"size:24" json:"kind"`
	Reason       string      `gorm:"type:text" json:"reason"`
	Evidence     string      `gorm:"type:text" json:"evidence"`
	Dismissed    bool        `json:"-"`
	TargetClue   *LivingClue `gorm:"foreignKey:TargetClueID" json:"target_clue,omitempty"`
}
