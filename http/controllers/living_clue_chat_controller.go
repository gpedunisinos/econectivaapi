package controllers

import (
	"fmt"
	"net/http"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/policies"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
)

// @Summary Get living clue chat messages
// @Description Get authenticated user chat history for a living clue
// @Tags living_clue_chat
// @Accept json
// @Produce json
// @Param id path int true "Living Clue ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} models.LivingClueChatMessage
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /living_clues/{id}/chat/messages [get]
// @Security Bearer
func FindLivingClueChatMessages(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	livingClue, err := repository.FindLivingClueByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Pista viva não encontrada!"})
		return
	}
	if !policies.NewLivingCluePolicy(&user).View(livingClue) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}
	limit, offset := getPaginationParams(context, 50)
	messages, err := repository.FindLivingClueChatMessagesByLivingClueAndUserID(livingClue.ID, user.ID, limit, offset)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	context.JSON(http.StatusOK, messages)
}

// @Summary Send living clue chat message
// @Description Send a message to a living clue chat, injecting living clue data, recent readings and chat history into the LLM context
// @Tags living_clue_chat
// @Accept json
// @Produce json
// @Param id path int true "Living Clue ID"
// @Param message body requests.SendLivingClueChatMessageInput true "Living clue chat message"
// @Success 200 {array} models.LivingClueChatMessage
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /living_clues/{id}/chat/messages [post]
// @Security Bearer
func SendLivingClueChatMessage(context *gin.Context) {
	context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, 32*1024)
	user := context.MustGet("user").(models.User)
	livingClue, err := repository.FindLivingClueByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Pista viva não encontrada!"})
		return
	}
	if !policies.NewLivingCluePolicy(&user).View(livingClue) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}
	var input requests.SendLivingClueChatMessageInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}
	readings, err := repository.FindLastSensorReadingsByLivingClueID(livingClue.ID, services.LivingClueReadingsWindowSize())
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	history, err := repository.FindLivingClueChatMessagesByLivingClueAndUserID(livingClue.ID, user.ID, services.LivingClueChatHistoryWindowSize(), 0)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	reply, err := services.GetChatCompletionService().GenerateReply(context.Request.Context(), services.ChatCompletionInput{
		User:         &user,
		PinnedClueID: livingClue.ID,
		Template:     services.ChatTemplateContext{Name: "living_clue_chat", SystemInstruction: services.BuildLivingClueSystemInstruction()},
		History:      services.FilterLivingClueChatHistoryForModel(history),
		Context:      services.BuildLivingClueChatContext(livingClue, readings),
		Message:      input.Message,
	})
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao consultar Gemini: " + err.Error()})
		return
	}
	tx := config.DB.Begin()
	userMessage := models.LivingClueChatMessage{LivingClueID: livingClue.ID, UserID: user.ID, Role: "user", Content: input.Message}
	if err := repository.CreateLivingClueChatMessage(tx, &userMessage); err != nil {
		tx.Rollback()
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	assistantMessage := models.LivingClueChatMessage{LivingClueID: livingClue.ID, UserID: user.ID, Role: "assistant", Content: reply}
	if err := repository.CreateLivingClueChatMessage(tx, &assistantMessage); err != nil {
		tx.Rollback()
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	if err := tx.Commit().Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	context.JSON(http.StatusOK, []models.LivingClueChatMessage{userMessage, assistantMessage})
}

// @Summary Delete living clue chat messages
// @Description Delete authenticated user chat history for a living clue
// @Tags living_clue_chat
// @Accept json
// @Produce json
// @Param id path int true "Living Clue ID"
// @Success 200 {object} models.APIMessage
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /living_clues/{id}/chat/messages [delete]
// @Security Bearer
func DeleteLivingClueChatMessages(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	livingClue, err := repository.FindLivingClueByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Pista viva não encontrada!"})
		return
	}
	if !policies.NewLivingCluePolicy(&user).View(livingClue) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}
	if err := repository.DeleteLivingClueChatMessagesByLivingClueAndUserID(livingClue.ID, user.ID); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	context.JSON(http.StatusOK, gin.H{"message": "Mensagens do chat da pista viva excluídas com sucesso"})
}
