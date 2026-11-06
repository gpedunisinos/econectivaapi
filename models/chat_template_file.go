package models

type ChatTemplateFile struct {
	DefaultModel
	ChatTemplateID   uint          `gorm:"not null;index" json:"chat_template_id"`
	ChatTemplate     *ChatTemplate `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"chat_template,omitempty"`
	Name             string        `gorm:"size:255;not null" binding:"required" json:"name"`
	MimeType         string        `gorm:"size:255;not null" json:"mime_type"`
	SourceType       string        `gorm:"size:20;not null;default:text" json:"source_type"`
	SourceURL        string        `gorm:"type:text" json:"source_url"`
	OriginalContent  string        `gorm:"type:longtext" json:"original_content"`
	ExtractedContent string        `gorm:"type:longtext" json:"extracted_content"`
	Content          string        `gorm:"type:longtext;not null" json:"content"`
	Status           string        `gorm:"size:20;not null;default:processed" json:"status"`
	ErrorMessage     string        `gorm:"type:text" json:"error_message"`
}
