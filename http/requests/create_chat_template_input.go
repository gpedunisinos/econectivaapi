package requests

type CreateChatTemplateInput struct {
	Name              string `json:"name" binding:"required"`
	Description       string `json:"description"`
	SystemInstruction string `json:"system_instruction" binding:"required"`
}
