package requests

import "github.com/OdairPianta/unisinos_usp_api/models"

type SendAIMessageInput struct {
	Message       string                  `json:"message" binding:"required,max=8000"`
	Context       string                  `json:"context" binding:"max=20000"`
	ScreenContext *models.AIScreenContext `json:"screen_context"`
}
