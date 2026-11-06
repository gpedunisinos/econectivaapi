package models

import "time"

// LockerItem holds a reference, never a copy or a public like.
type LockerItem struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"uniqueIndex:locker_identity;not null" json:"-"`
	EntityType string    `gorm:"size:32;uniqueIndex:locker_identity;not null" json:"entity_type"`
	EntityID   uint      `gorm:"uniqueIndex:locker_identity;not null" json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
	Name       string    `gorm:"-" json:"name"`
	Origin     string    `gorm:"-" json:"origin,omitempty"`
	Available  bool      `gorm:"-" json:"available"`
}

type Participant struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ClueComment struct {
	ReplyToAuthor *Participant `gorm:"-" json:"reply_to_author,omitempty"`
	ID            uint         `gorm:"primaryKey" json:"id"`
	LivingClueID  uint         `gorm:"index;not null" json:"living_clue_id"`
	UserID        uint         `gorm:"index;not null" json:"user_id"`
	Author        Participant  `gorm:"-" json:"author"`
	RootID        *uint        `gorm:"index" json:"root_id"`
	ReplyToID     *uint        `json:"reply_to_id"`
	Content       string       `gorm:"type:text" json:"content"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	EditedAt      *time.Time   `json:"edited_at"`
	RemovedAt     *time.Time   `json:"removed_at"`
}

type ClueEdit struct {
	EventType    string      `gorm:"size:16;not null;default:updated" json:"event_type"`
	ID           uint        `gorm:"primaryKey" json:"id"`
	LivingClueID uint        `gorm:"index;not null" json:"living_clue_id"`
	UserID       uint        `json:"-"`
	Author       Participant `gorm:"-" json:"author"`
	CreatedAt    time.Time   `json:"created_at"`
	Description  string      `gorm:"-" json:"description"`
}

type LinkPreview struct {
	URL         string    `gorm:"primaryKey;size:768" json:"url"`
	Title       string    `gorm:"type:text" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Image       string    `gorm:"type:text" json:"image"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Derived references associate the shared metadata cache with each clue's URLs.
// A hash avoids truncating long URLs to fit a composite database index.
type CluePreviewReference struct {
	LivingClueID uint   `gorm:"primaryKey" json:"living_clue_id"`
	URLHash      string `gorm:"primaryKey;size:64" json:"-"`
	URL          string `gorm:"type:text;not null" json:"url"`
}
