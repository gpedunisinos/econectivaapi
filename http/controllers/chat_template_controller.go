package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/policies"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
)

// @Summary Get chat templates
// @Param search query string false "Search agent name or description"
// @Param limit query int false "Page size (maximum 200)" default(50)
// @Param offset query int false "Pagination offset" default(0)
// @Description Get all chat templates available to authenticated users
// @Tags chat_templates
// @Accept json
// @Produce json
// @Success 200 {array} models.ChatTemplate
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /chat_templates [get]
// @Security Bearer
func FindChatTemplates(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewChatTemplatePolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	limit, offset := getPaginationParams(context, 50)
	templates, err := repository.FindChatTemplates(limit, offset, context.Query("search"))
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, templates)
}

// @Summary Get a registered agent by ID
// @Description Get an existing agent available to the authenticated user, including agents linked to clues.
// @Tags chat_templates
// @Produce json
// @Param id path int true "Agent ID"
// @Success 200 {object} models.ChatTemplate
// @Failure 400 {object} models.APIMessage
// @Failure 401 {object} models.APIMessage
// @Failure 404 {object} models.APIMessage
// @Router /chat_templates/{id} [get]
// @Security Bearer
func FindChatTemplateByID(context *gin.Context) {
	id, err := strconv.ParseUint(context.Param("id"), 10, 64)
	if err != nil || id == 0 {
		context.JSON(400, gin.H{"message": "Agente inválido."})
		return
	}
	template, err := repository.FindChatTemplateByID(context.Param("id"))
	if err != nil {
		context.JSON(404, gin.H{"message": "Agente não encontrado ou indisponível."})
		return
	}
	user := context.MustGet("user").(models.User)
	if !policies.NewChatTemplatePolicy(&user).View(template) {
		context.JSON(401, gin.H{"message": "Você não tem permissão para acessar este agente."})
		return
	}
	context.JSON(200, template)
}

// @Summary Create chat template
// @Description Create chat template with embedded context files
// @Tags chat_templates
// @Accept json
// @Produce json
// @Param chat_template body requests.CreateChatTemplateInput true "Chat template"
// @Success 200 {object} models.ChatTemplate
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /chat_templates [post]
// @Security Bearer
func CreateChatTemplate(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewChatTemplatePolicy(&user).Create() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.CreateChatTemplateInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	template := models.ChatTemplate{
		Name:              input.Name,
		Description:       input.Description,
		SystemInstruction: input.SystemInstruction,
		UserID:            user.ID,
	}

	if err := repository.CreateChatTemplate(&template); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, template)
}

// @Summary Update chat template
// @Description Update chat template and replace embedded context files (owner or administrator)
// @Tags chat_templates
// @Accept json
// @Produce json
// @Param id path int true "ChatTemplate ID"
// @Param chat_template body requests.CreateChatTemplateInput true "Chat template"
// @Success 200 {object} models.ChatTemplate
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /chat_templates/{id} [put]
// @Security Bearer
func UpdateChatTemplate(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	template, err := repository.FindChatTemplateByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Template de chat não encontrado"})
		return
	}

	if !policies.NewChatTemplatePolicy(&user).Update(template) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.CreateChatTemplateInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	tx := config.DB.Begin()
	template.Name = input.Name
	template.Description = input.Description
	template.SystemInstruction = input.SystemInstruction

	if err := tx.Model(template).Updates(map[string]interface{}{
		"name":               template.Name,
		"description":        template.Description,
		"system_instruction": template.SystemInstruction,
	}).Error; err != nil {
		tx.Rollback()
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	if err := tx.Commit().Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	updatedTemplate, err := repository.FindChatTemplateByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, updatedTemplate)
}

// @Summary Delete chat template
// @Description Delete chat template (owner or administrator)
// @Tags chat_templates
// @Accept json
// @Produce json
// @Param id path int true "ChatTemplate ID"
// @Success 200 {object} models.APIMessage
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /chat_templates/{id} [delete]
// @Security Bearer
func DeleteChatTemplate(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	template, err := repository.FindChatTemplateByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Template de chat não encontrado"})
		return
	}

	if !policies.NewChatTemplatePolicy(&user).Delete(template) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	if err := repository.DeleteChatTemplate(template); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Template de chat excluído com sucesso"})
}

// @Summary Get chat messages
// @Description Get authenticated user chat history by template, allowing access to any existing template
// @Tags chat_messages
// @Accept json
// @Produce json
// @Param id path int true "ChatTemplate ID"
// @Success 200 {array} models.ChatMessage
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /chat_templates/{id}/messages [get]
// @Security Bearer
func FindChatMessages(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	template, err := repository.FindChatTemplateByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Template de chat não encontrado"})
		return
	}

	if !policies.NewChatTemplatePolicy(&user).View(template) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	limit, offset := getPaginationParams(context, 50)
	messages, err := repository.FindChatMessagesByTemplateAndUserID(template.ID, user.ID, limit, offset)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, messages)
}

// @Summary Delete chat messages
// @Description Delete authenticated user chat history by template, allowing access to any existing template
// @Tags chat_messages
// @Accept json
// @Produce json
// @Param id path int true "ChatTemplate ID"
// @Success 200 {object} models.APIMessage
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /chat_templates/{id}/messages [delete]
// @Security Bearer
func DeleteChatMessages(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	template, err := repository.FindChatTemplateByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Template de chat não encontrado"})
		return
	}

	if !policies.NewChatTemplatePolicy(&user).View(template) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	if err := repository.DeleteChatMessagesByTemplateAndUserID(template.ID, user.ID); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Mensagens do chat excluídas com sucesso"})
}

// @Summary Send chat message
// @Description Send chat message using a template and persist user and assistant messages
// @Tags chat_messages
// @Accept json
// @Produce json
// @Param id path int true "ChatTemplate ID"
// @Param message body requests.SendChatMessageInput true "Message"
// @Success 200 {array} models.ChatMessage
// @Failure 400 {object} models.APIMessage "Bad request"
// @Router /chat_templates/{id}/messages [post]
// @Security Bearer
func SendChatMessage(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	template, err := repository.FindChatTemplateByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Template de chat não encontrado"})
		return
	}

	if !policies.NewChatTemplatePolicy(&user).View(template) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.SendChatMessageInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	history, err := repository.FindChatMessagesByTemplateAndUserID(template.ID, user.ID, 100, 0)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	reply, err := services.GetChatCompletionService().GenerateReply(
		context.Request.Context(),
		services.ChatCompletionInput{
			Template: services.BuildChatTemplateContext(*template),
			History:  history,
			Context:  services.BuildChatTemplateContextText(*template),
			Message:  input.Message,
		},
	)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Erro ao consultar Gemini: " + err.Error()})
		return
	}

	tx := config.DB.Begin()
	userMessage := models.ChatMessage{
		ChatTemplateID: template.ID,
		UserID:         user.ID,
		Role:           "user",
		Content:        input.Message,
	}
	if err := repository.CreateChatMessage(tx, &userMessage); err != nil {
		tx.Rollback()
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	assistantMessage := models.ChatMessage{
		ChatTemplateID: template.ID,
		UserID:         user.ID,
		Role:           "assistant",
		Content:        reply,
	}
	if err := repository.CreateChatMessage(tx, &assistantMessage); err != nil {
		tx.Rollback()
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	if err := tx.Commit().Error; err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, []models.ChatMessage{userMessage, assistantMessage})
}

func getPaginationParams(context *gin.Context, defaultLimit int) (int, int) {
	limit := defaultLimit
	offset := 0

	if rawLimit := context.Query("limit"); rawLimit != "" {
		if parsedLimit, err := strconv.Atoi(rawLimit); err == nil && parsedLimit > 0 {
			if parsedLimit > 200 {
				parsedLimit = 200
			}
			limit = parsedLimit
		}
	}

	if rawOffset := context.Query("offset"); rawOffset != "" {
		if parsedOffset, err := strconv.Atoi(rawOffset); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	return limit, offset
}
