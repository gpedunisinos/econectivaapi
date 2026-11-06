package requests

type CreateChatTemplateFileInput struct {
	ChatTemplateID uint   `json:"chat_template_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	MimeType       string `json:"mime_type"`
	Type           string `json:"type"`
	URL            string `json:"url"`
	Content        string `json:"content"`
}
