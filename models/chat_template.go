package models

type ChatTemplate struct {
	DefaultModel
	Name              string             `gorm:"size:255;not null" binding:"required" json:"name"`
	Description       string             `gorm:"type:text" json:"description"`
	SystemInstruction string             `gorm:"type:longtext;not null" binding:"required" json:"system_instruction"`
	UserID            uint               `gorm:"not null;index" json:"user_id"`
	User              *User              `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	ChatTemplateFiles []ChatTemplateFile `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"chat_template_files,omitempty"`
}
