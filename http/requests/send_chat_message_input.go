package requests

type SendChatMessageInput struct {
	Message string `json:"message" binding:"required"`
}
