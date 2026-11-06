package models

type ChatMessage struct {
	DefaultModel
	ChatTemplateID uint          `gorm:"not null;index" json:"chat_template_id"`
	ChatTemplate   *ChatTemplate `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"chat_template,omitempty"`
	UserID         uint          `gorm:"not null;index" json:"user_id"`
	User           *User         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	Role           string        `gorm:"size:20;not null" binding:"required" json:"role"`
	Content        string        `gorm:"type:longtext;not null" binding:"required" json:"content"`
}
