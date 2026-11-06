package controllers

import (
	"fmt"
	"net/http"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
)

// @Summary Get AI messages
// @Description Get authenticated user AI chat history
// @Tags ai_messages
// @Accept json
// @Produce json
// @Success 200 {array} models.AIMessage
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /ai/messages [get]
// @Security Bearer
func FindAIMessages(context *gin.Context) {
	user := context.MustGet("user").(models.User)

	limit, offset := getPaginationParams(context, 50)
	messages, err := repository.FindAIMessagesByUserID(user.ID, limit, offset)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, messages)
}

// @Summary Send AI message
// @Description Conversa global com contexto estruturado atual, consultas MCP autorizadas e histórico próprio por usuário. O contexto do cliente nunca concede autorização.
// @Tags ai_messages
// @Accept json
// @Produce json
// @Param message body requests.SendAIMessageInput true "AI message"
// @Success 200 {array} models.AIMessage
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /ai/messages [post]
// @Security Bearer
func SendAIMessage(context *gin.Context) {
	context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, 128*1024)
	user := context.MustGet("user").(models.User)

	var input requests.SendAIMessageInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	history, err := repository.FindAIMessagesByUserID(user.ID, services.AIHistoryWindowSize(), 0)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	currentContext := services.BuildCurrentScreenContext(input.ScreenContext, input.Context, &user)
	reply, err := services.GetChatCompletionService().GenerateReply(
		context.Request.Context(),
		services.ChatCompletionInput{
			Template: services.ChatTemplateContext{
				Name:              "ai_messages",
				SystemInstruction: services.BuildAIMessageSystemInstruction(),
			},
			History:       services.FilterAIHistoryForModel(history),
			Context:       currentContext,
			User:          &user,
			ScreenContext: input.ScreenContext,
			Message:       input.Message,
		},
	)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	tx := config.DB.Begin()
	userMessage := models.AIMessage{
		UserID:  user.ID,
		Role:    "user",
		Content: input.Message,
		Context: currentContext,
	}
	if err := repository.CreateAIMessage(tx, &userMessage); err != nil {
		tx.Rollback()
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	assistantMessage := models.AIMessage{
		UserID:  user.ID,
		Role:    "assistant",
		Content: reply,
		Context: services.BuildAIMessageAssistantContext(currentContext),
	}
	if err := repository.CreateAIMessage(tx, &assistantMessage); err != nil {
		tx.Rollback()
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	if err := tx.Commit().Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, []models.AIMessage{userMessage, assistantMessage})
}
