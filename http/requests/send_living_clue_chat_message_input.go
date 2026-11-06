package requests

type SendLivingClueChatMessageInput struct {
	Message string `json:"message" binding:"required,max=8000"`
}
