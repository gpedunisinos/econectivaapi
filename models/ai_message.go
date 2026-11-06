package models

type AIMessage struct {
	DefaultModel
	UserID  uint   `gorm:"not null;index" json:"user_id"`
	User    *User  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	Role    string `gorm:"size:20;not null" binding:"required" json:"role"`
	Content string `gorm:"type:longtext;not null" binding:"required" json:"content"`
	Context string `gorm:"type:longtext" json:"context"`
}
