package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ParticipantConversationInput struct {
	Participant string `json:"participant" binding:"required"`
}
type ParticipantTextInput struct {
	Text     string `json:"text"`
	ClientID string `json:"client_id"`
}
type ParticipantReadInput struct {
	Through uint64 `json:"through"`
}
type ParticipantReportInput struct {
	Description string `json:"description"`
}
type ParticipantConversationView struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Online      bool      `json:"online"`
	Activity    string    `json:"activity"`
	Preview     string    `json:"preview"`
	Unread      int64     `json:"unread"`
	Blocked     bool      `json:"blocked"`
	BlockedByMe bool      `json:"blocked_by_me"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func participantConversationView(c models.ParticipantConversation, user uint) ParticipantConversationView {
	peer, read := c.UserA, c.ReadB
	if peer == user {
		peer = c.UserB
		read = c.ReadA
	}
	view := ParticipantConversationView{ID: c.ID, Name: "Participante indisponível", Activity: "Fora do espaço", UpdatedAt: c.UpdatedAt}
	var blocks []models.ParticipantConversationBlock
	if config.DB.Where("conversation_id = ?", c.ID).Find(&blocks).Error == nil {
		view.Blocked = len(blocks) > 0
		for _, block := range blocks {
			view.BlockedByMe = view.BlockedByMe || block.UserID == user
		}
	}
	var target models.User
	if config.DB.First(&target, peer).Error == nil {
		name := strings.Fields(target.Name)
		if len(name) > 0 {
			runes := []rune(name[0])
			if len(runes) > 40 {
				runes = runes[:40]
			}
			view.Name = string(runes)
		}
	}
	if p, ok := presenceHub.PersonForUser(peer); ok {
		view.Name = p.Name
		view.Online = true
		view.Activity = p.Activity
	}
	if view.Blocked && !view.BlockedByMe {
		return view
	}
	var last models.ParticipantMessage
	if config.DB.Where("conversation_id = ?", c.ID).Order("id DESC").First(&last).Error == nil {
		runes := []rune(last.Text)
		if len(runes) > 100 {
			runes = runes[:100]
		}
		view.Preview = string(runes)
	}
	config.DB.Model(&models.ParticipantMessage{}).Where("conversation_id = ? AND sender_id <> ? AND id > ?", c.ID, user, read).Count(&view.Unread)
	return view
}
func participantActor(c *gin.Context) uint {
	c.Header("Cache-Control", "no-store")
	return c.MustGet("user").(models.User).ID
}
func participantConversation(c *gin.Context) (models.ParticipantConversation, uint, bool) {
	user := participantActor(c)
	conversation, err := services.FindParticipantConversation(c.Param("id"), user)
	if err != nil {
		c.JSON(404, gin.H{"message": "Conversa não encontrada."})
		return conversation, user, false
	}
	return conversation, user, true
}

// ListParticipantConversations godoc
// @Summary Conversas privadas da conta autenticada
// @Tags participant_chat
// @Produce json
// @Security Bearer
// @Param offset query int false "Deslocamento" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} models.APIMessage
// @Router /me/conversations [get]
func ListParticipantConversations(c *gin.Context) {
	user := participantActor(c)
	offset, _ := strconv.Atoi(c.Query("offset"))
	if offset < 0 || offset > 100000 {
		offset = 0
	}
	rows := []models.ParticipantConversation{}
	err := config.DB.Where("user_a = ? OR user_b = ?", user, user).Order("updated_at DESC, id ASC").Limit(31).Offset(offset).Find(&rows).Error
	if err != nil {
		c.JSON(503, gin.H{"message": "Não foi possível carregar as conversas."})
		return
	}
	more := len(rows) > 30
	if more {
		rows = rows[:30]
	}
	views := []ParticipantConversationView{}
	for _, row := range rows {
		views = append(views, participantConversationView(row, user))
	}
	var unread int64
	config.DB.Raw(`SELECT COUNT(*) FROM participant_messages m JOIN participant_conversations c ON m.conversation_id = c.id WHERE NOT EXISTS (SELECT 1 FROM participant_conversation_blocks b WHERE b.conversation_id = c.id AND b.user_id <> ?) AND ((c.user_a = ? AND m.sender_id <> ? AND m.id > c.read_a) OR (c.user_b = ? AND m.sender_id <> ? AND m.id > c.read_b))`, user, user, user, user, user).Scan(&unread)
	c.JSON(200, gin.H{"conversations": views, "has_more": more, "unread": unread})
}

// OpenParticipantConversation godoc
// @Summary Conectar conversa a um participante do espaço
// @Description Usa somente o identificador opaco da presença. Identidade do remetente vem da sessão. Reutiliza a conversa privada entre as duas contas.
// @Tags participant_chat
// @Accept json
// @Produce json
// @Security Bearer
// @Param input body ParticipantConversationInput true "Participante"
// @Success 200 {object} ParticipantConversationView
// @Failure 400 {object} models.APIMessage
// @Failure 404 {object} models.APIMessage
// @Router /me/conversations [post]
func OpenParticipantConversation(c *gin.Context) {
	user := participantActor(c)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	var input ParticipantConversationInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(400, gin.H{"message": "Selecione um participante."})
		return
	}
	peer, ok := presenceHub.ResolveParticipant(input.Participant)
	if !ok || peer == user {
		c.JSON(404, gin.H{"message": "Esse participante não está disponível. Atualize a lista de naves."})
		return
	}
	conversation, err := services.OpenParticipantConversation(user, peer)
	if err != nil {
		c.JSON(503, gin.H{"message": "Não foi possível abrir a conversa."})
		return
	}
	c.JSON(200, participantConversationView(conversation, user))
}

// GetParticipantMessages godoc
// @Summary Ler texto e emojis de uma conversa privada
// @Tags participant_chat
// @Produce json
// @Security Bearer
// @Param id path string true "Conversa"
// @Param before query int false "Mensagens anteriores ao ID"
// @Success 200 {object} services.ParticipantMessagePage
// @Failure 404 {object} models.APIMessage
// @Router /me/conversations/{id}/messages [get]
func GetParticipantMessages(c *gin.Context) {
	conversation, user, ok := participantConversation(c)
	if !ok {
		return
	}
	var peerBlock int64
	if err := config.DB.Model(&models.ParticipantConversationBlock{}).Where("conversation_id = ? AND user_id <> ?", conversation.ID, user).Count(&peerBlock).Error; err != nil {
		c.JSON(503, gin.H{"message": "Não foi possível verificar o bloqueio."})
		return
	}
	if peerBlock > 0 {
		c.JSON(403, gin.H{"message": "O participante bloqueou esta conversa."})
		return
	}
	before, _ := strconv.ParseUint(c.Query("before"), 10, 64)
	page, err := services.ParticipantMessages(conversation, user, before)
	if err != nil {
		c.JSON(503, gin.H{"message": "Não foi possível carregar as mensagens."})
		return
	}
	c.JSON(200, page)
}

// SendParticipantMessage godoc
// @Summary Enviar texto e emojis a outro participante
// @Description Até 2000 caracteres; client_id UUID evita duplicação em novas tentativas. Nenhum conteúdo é enviado à cena espacial ou à IAraeté.
// @Tags participant_chat
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "Conversa"
// @Param input body ParticipantTextInput true "Mensagem"
// @Success 200 {object} services.ParticipantMessageView
// @Failure 400 {object} models.APIMessage
// @Failure 404 {object} models.APIMessage
// @Failure 409 {object} models.APIMessage
// @Failure 429 {object} models.APIMessage
// @Router /me/conversations/{id}/messages [post]
func SendParticipantMessage(c *gin.Context) {
	conversation, user, ok := participantConversation(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16384)
	var input ParticipantTextInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(400, gin.H{"message": "Informe uma mensagem válida."})
		return
	}
	message, created, err := services.SendParticipantMessage(conversation, user, input.Text, input.ClientID)
	if err != nil {
		status := 503
		switch {
		case errors.Is(err, services.ErrParticipantText):
			status = 400
		case errors.Is(err, services.ErrParticipantRetry):
			status = 409
		case errors.Is(err, services.ErrParticipantRate):
			status = 429
		case errors.Is(err, services.ErrParticipantBlocked):
			status = 403
		case errors.Is(err, gorm.ErrRecordNotFound):
			status = 404
		}
		c.JSON(status, gin.H{"message": map[bool]string{true: err.Error(), false: "Não foi possível enviar a mensagem."}[status != 503]})
		return
	}
	if created {
		presenceHub.NotifyConversation(conversation.UserA, conversation.UserB, conversation.ID)
	}
	c.JSON(200, services.ParticipantMessageDTO(message, user))
}

// ReportParticipantConversation godoc
// @Summary Denunciar uma conversa privada
// @Tags participant_chat
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "Conversa"
// @Param input body ParticipantReportInput true "Descrição opcional"
// @Success 201 {object} models.APIMessage
// @Failure 400 {object} models.APIMessage
// @Failure 404 {object} models.APIMessage
// @Router /me/conversations/{id}/report [post]
func ReportParticipantConversation(c *gin.Context) {
	conversation, user, ok := participantConversation(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	var input ParticipantReportInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(400, gin.H{"message": "Denúncia inválida."})
		return
	}
	if err := services.ReportParticipantConversation(conversation, user, input.Description); err != nil {
		if errors.Is(err, services.ErrParticipantDescription) {
			c.JSON(400, gin.H{"message": err.Error()})
		} else {
			c.JSON(503, gin.H{"message": "Não foi possível registrar a denúncia."})
		}
		return
	}
	c.JSON(201, gin.H{"message": "Denúncia registrada."})
}

// BlockParticipantConversation godoc
// @Summary Bloquear o outro participante da conversa
// @Tags participant_chat
// @Produce json
// @Security Bearer
// @Param id path string true "Conversa"
// @Success 200 {object} models.APIMessage
// @Failure 404 {object} models.APIMessage
// @Router /me/conversations/{id}/block [post]
func BlockParticipantConversation(c *gin.Context) {
	conversation, user, ok := participantConversation(c)
	if !ok {
		return
	}
	if services.BlockParticipantConversation(conversation, user) != nil {
		c.JSON(503, gin.H{"message": "Não foi possível bloquear a conversa."})
		return
	}
	c.JSON(200, gin.H{"message": "Conversa bloqueada."})
}

// UnblockParticipantConversation godoc
// @Summary Desbloquear o outro participante da conversa
// @Tags participant_chat
// @Produce json
// @Security Bearer
// @Param id path string true "Conversa"
// @Success 200 {object} models.APIMessage
// @Failure 404 {object} models.APIMessage
// @Router /me/conversations/{id}/block [delete]
func UnblockParticipantConversation(c *gin.Context) {
	conversation, user, ok := participantConversation(c)
	if !ok {
		return
	}
	if services.UnblockParticipantConversation(conversation, user) != nil {
		c.JSON(503, gin.H{"message": "Não foi possível desbloquear a conversa."})
		return
	}
	c.JSON(200, gin.H{"message": "Conversa desbloqueada."})
}

// ReadParticipantMessages godoc
// @Summary Atualizar leitura privada da conversa
// @Tags participant_chat
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "Conversa"
// @Param input body ParticipantReadInput true "Última mensagem exibida"
// @Success 200 {object} models.APIMessage
// @Failure 404 {object} models.APIMessage
// @Router /me/conversations/{id}/read [post]
func ReadParticipantMessages(c *gin.Context) {
	conversation, user, ok := participantConversation(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	var input ParticipantReadInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(400, gin.H{"message": "Leitura inválida."})
		return
	}
	if services.ReadParticipantConversation(conversation, user, input.Through) != nil {
		c.JSON(503, gin.H{"message": "Não foi possível atualizar a leitura."})
		return
	}

	c.JSON(200, gin.H{"message": "Leitura atualizada."})
}
