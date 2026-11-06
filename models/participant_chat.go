package models

import "time"

// These records are private to the two accounts. Public APIs use explicit DTOs.
type ParticipantConversation struct {
	ID        string    `gorm:"primaryKey;size:36" json:"-"`
	UserA     uint      `gorm:"not null;uniqueIndex:participant_pair;index" json:"-"`
	UserB     uint      `gorm:"not null;uniqueIndex:participant_pair;index" json:"-"`
	ReadA     uint64    `json:"-"`
	ReadB     uint64    `json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `gorm:"index" json:"-"`
}
type ParticipantMessage struct {
	ID             uint64    `gorm:"primaryKey" json:"-"`
	ConversationID string    `gorm:"size:36;not null;index;uniqueIndex:participant_message_retry" json:"-"`
	SenderID       uint      `gorm:"not null;uniqueIndex:participant_message_retry" json:"-"`
	ClientID       string    `gorm:"size:36;not null;uniqueIndex:participant_message_retry" json:"-"`
	Text           string    `gorm:"type:text;not null" json:"-"`
	CreatedAt      time.Time `json:"-"`
}

type ParticipantConversationReport struct {
	ID             uint64    `gorm:"primaryKey" json:"-"`
	UserID         uint      `gorm:"not null;index" json:"-"`
	ConversationID string    `gorm:"size:36;not null;index" json:"-"`
	Description    *string   `gorm:"type:text" json:"-"`
	CreatedAt      time.Time `json:"-"`
}

type ParticipantConversationBlock struct {
	ID             uint64    `gorm:"primaryKey" json:"-"`
	UserID         uint      `gorm:"not null;uniqueIndex:participant_block" json:"-"`
	ConversationID string    `gorm:"size:36;not null;uniqueIndex:participant_block" json:"-"`
	CreatedAt      time.Time `json:"-"`
}
