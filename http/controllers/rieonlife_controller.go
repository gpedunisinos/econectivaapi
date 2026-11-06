package controllers

import (
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/policies"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func currentUser(c *gin.Context) models.User { return c.MustGet("user").(models.User) }
func rieonError(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, gin.H{"message": message})
}
func dbResponse(c *gin.Context, err error, value interface{}) {
	if err != nil {
		rieonError(c, 500, "Não foi possível concluir a operação.")
		return
	}
	c.JSON(200, value)
}

// GetPreferences godoc
// @Summary Read the authenticated account preference
// @Tags preferences
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /me/preferences [get]
// @Security Bearer
func GetPreferences(c *gin.Context) {
	c.JSON(200, gin.H{"usage_mode": currentUser(c).UsageMode, "profile": currentUser(c).Profile})
}

// SavePreferences godoc
// @Summary Choose explorer or inventor for the authenticated account
// @Tags preferences
// @Accept json
// @Produce json
// @Param input body map[string]string true "usage_mode: explorer or inventor"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.APIMessage
// @Router /me/preferences [put]
// @Security Bearer
func SavePreferences(c *gin.Context) {
	var in struct {
		Mode string `json:"usage_mode"`
	}
	if c.ShouldBindJSON(&in) != nil || (in.Mode != "explorer" && in.Mode != "inventor") {
		rieonError(c, 400, "Escolha Explorador ou Inventor.")
		return
	}
	u := currentUser(c)
	dbResponse(c, config.DB.Model(&models.User{}).Where("id = ?", u.ID).Update("usage_mode", in.Mode).Error, gin.H{"usage_mode": in.Mode})
}

func lockerTarget(c *gin.Context, typ string, id uint) (string, string, bool) {
	u := currentUser(c)
	if typ == "living_clue" {
		clue, err := repository.FindLivingClueReferenceByID(strconv.Itoa(int(id)))
		if err == nil && policies.NewLivingCluePolicy(&u).View(clue) {
			return clue.Name, "Pista", true
		}
	}
	if typ == "node" {
		var node models.Node
		if config.DB.First(&node, id).Error == nil && policies.NewNodePolicy(&u).View(&node) {
			return node.Name, "Problemática", true
		}
	}
	if typ == "agent" {
		var agent models.ChatTemplate
		if config.DB.First(&agent, id).Error == nil && policies.NewChatTemplatePolicy(&u).View(&agent) {
			return agent.Name, "Agente", true
		}
	}
	return "Conteúdo indisponível", "", false
}
func lockerIdentity(c *gin.Context) (string, uint, bool) {
	typ := c.Param("type")
	id, err := strconv.ParseUint(c.Param("entity"), 10, 32)
	if err != nil || id == 0 || (typ != "living_clue" && typ != "node" && typ != "agent") {
		rieonError(c, 400, "Referência inválida.")
		return "", 0, false
	}
	return typ, uint(id), true
}

// ListLocker godoc
// @Summary List personal saved references, newest first; unavailable targets disclose no content
// @Tags locker
// @Produce json
// @Success 200 {array} models.LockerItem
// @Router /me/locker [get]
// @Security Bearer
func ListLocker(c *gin.Context) {
	items := []models.LockerItem{}
	err := config.DB.Where("user_id = ?", currentUser(c).ID).Order("created_at DESC, id DESC").Find(&items).Error
	for i := range items {
		items[i].Name, items[i].Origin, items[i].Available = lockerTarget(c, items[i].EntityType, items[i].EntityID)
	}
	dbResponse(c, err, items)
}

// SaveLocker godoc
// @Summary Idempotently save a personal reference
// @Tags locker
// @Param type path string true "living_clue, node or agent"
// @Param entity path int true "Entity ID"
// @Success 200 {object} models.LockerItem
// @Router /me/locker/{type}/{entity} [put]
// @Security Bearer
func SaveLocker(c *gin.Context) {
	typ, id, ok := lockerIdentity(c)
	if !ok {
		return
	}
	name, origin, available := lockerTarget(c, typ, id)
	if !available {
		rieonError(c, 404, "Conteúdo indisponível.")
		return
	}
	item := models.LockerItem{UserID: currentUser(c).ID, EntityType: typ, EntityID: id}
	err := config.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&item).Error
	if err == nil {
		err = config.DB.Where("user_id = ? AND entity_type = ? AND entity_id = ?", item.UserID, typ, id).First(&item).Error
	}
	item.Name = name
	item.Origin = origin
	item.Available = true
	dbResponse(c, err, item)
}

// RemoveLocker godoc
// @Summary Remove only the authenticated account's saved reference
// @Tags locker
// @Param type path string true "Entity type"
// @Param entity path int true "Entity ID"
// @Success 200 {object} map[string]bool
// @Router /me/locker/{type}/{entity} [delete]
// @Security Bearer
func RemoveLocker(c *gin.Context) {
	typ, id, ok := lockerIdentity(c)
	if !ok {
		return
	}
	dbResponse(c, config.DB.Where("user_id = ? AND entity_type = ? AND entity_id = ?", currentUser(c).ID, typ, id).Delete(&models.LockerItem{}).Error, gin.H{"removed": true})
}

func accessibleClue(c *gin.Context) *models.LivingClue {
	clue, err := repository.FindLivingClueReferenceByID(c.Param("id"))
	if err != nil {
		rieonError(c, 404, "Pista não encontrada.")
		return nil
	}
	u := currentUser(c)
	if !policies.NewLivingCluePolicy(&u).View(clue) {
		rieonError(c, 403, "Acesso negado.")
		return nil
	}
	return clue
}
func page(c *gin.Context) (int, int) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
func author(id uint) models.Participant {
	var u models.User
	config.DB.Select("id", "name").First(&u, id)
	if u.Name == "" {
		return models.Participant{ID: id, Name: "Participante"}
	}
	return models.Participant{ID: u.ID, Name: u.Name}
}

// ListComments godoc
// @Summary Paginated roots (newest first) or replies (oldest first) in a clue
// @Tags compartilhacoes
// @Param id path int true "Clue ID"
// @Param root_id query int false "Thread ID; omit for roots"
// @Param limit query int false "Page size (1-100)"
// @Param offset query int false "Offset"
// @Success 200 {array} models.ClueComment
// @Router /living_clues/{id}/comments [get]
// @Security Bearer
func ListComments(c *gin.Context) {
	clue := accessibleClue(c)
	if clue == nil {
		return
	}
	limit, offset := page(c)
	q := config.DB.Where("living_clue_id = ?", clue.ID)
	if root := c.Query("root_id"); root != "" {
		q = q.Where("root_id = ?", root).Order("created_at ASC, id ASC")
	} else {
		q = q.Where("root_id IS NULL").Order("created_at DESC, id DESC")
	}
	items := []models.ClueComment{}
	err := q.Limit(limit).Offset(offset).Find(&items).Error
	for i := range items {
		items[i].Author = author(items[i].UserID)
		if items[i].ReplyToID != nil {
			var target models.ClueComment
			if config.DB.Select("user_id").Where("id = ? AND living_clue_id = ?", *items[i].ReplyToID, clue.ID).First(&target).Error == nil {
				participant := author(target.UserID)
				items[i].ReplyToAuthor = &participant
			}
		}
	}
	dbResponse(c, err, items)
}
func commentBody(c *gin.Context) (string, *uint, bool) {
	var in struct {
		Content string `json:"content"`
		Reply   *uint  `json:"reply_to_id"`
	}
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.Content) == "" || utf8.RuneCountInString(in.Content) > 5000 {
		rieonError(c, 400, "Escreva uma mensagem de até 5000 caracteres.")
		return "", nil, false
	}
	return in.Content, in.Reply, true
}

// CreateComment godoc
// @Summary Publish text or emoji; replies always belong to one root thread
// @Tags compartilhacoes
// @Param id path int true "Clue ID"
// @Param input body map[string]interface{} true "content and optional reply_to_id"
// @Success 200 {object} models.ClueComment
// @Router /living_clues/{id}/comments [post]
// @Security Bearer
func CreateComment(c *gin.Context) {
	clue := accessibleClue(c)
	if clue == nil {
		return
	}
	content, reply, ok := commentBody(c)
	if !ok {
		return
	}
	item := models.ClueComment{LivingClueID: clue.ID, UserID: currentUser(c).ID, Content: content, ReplyToID: reply}
	if reply != nil {
		var target models.ClueComment
		if config.DB.Where("id = ? AND living_clue_id = ?", *reply, clue.ID).First(&target).Error != nil {
			rieonError(c, 400, "Resposta inválida.")
			return
		}
		root := target.ID
		if target.RootID != nil {
			root = *target.RootID
		}
		item.RootID = &root
	}
	err := config.DB.Create(&item).Error
	item.Author = author(item.UserID)
	dbResponse(c, err, item)
}
func ownComment(c *gin.Context) *models.ClueComment {
	clue := accessibleClue(c)
	if clue == nil {
		return nil
	}
	var item models.ClueComment
	if config.DB.Where("id = ? AND living_clue_id = ?", c.Param("comment"), clue.ID).First(&item).Error != nil {
		rieonError(c, 404, "Mensagem não encontrada.")
		return nil
	}
	if item.UserID != currentUser(c).ID {
		rieonError(c, 403, "Somente o autor pode alterar esta mensagem.")
		return nil
	}
	if item.RemovedAt != nil {
		rieonError(c, 400, "Comentário excluído.")
		return nil
	}
	return &item
}

// UpdateComment godoc
// @Summary Edit only your own message, preserving publication order
// @Tags compartilhacoes
// @Param id path int true "Clue ID"
// @Param comment path int true "Comment ID"
// @Param input body map[string]string true "content"
// @Success 200 {object} models.ClueComment
// @Router /living_clues/{id}/comments/{comment} [put]
// @Security Bearer
func UpdateComment(c *gin.Context) {
	item := ownComment(c)
	if item == nil {
		return
	}
	content, _, ok := commentBody(c)
	if !ok {
		return
	}
	now := time.Now()
	err := config.DB.Model(item).Updates(map[string]interface{}{"content": content, "edited_at": now}).Error
	item.Content = content
	item.EditedAt = &now
	item.Author = author(item.UserID)
	dbResponse(c, err, item)
}

// DeleteComment godoc
// @Summary Replace own message with a tombstone, preserving replies
// @Tags compartilhacoes
// @Param id path int true "Clue ID"
// @Param comment path int true "Comment ID"
// @Success 200 {object} map[string]bool
// @Router /living_clues/{id}/comments/{comment} [delete]
// @Security Bearer
func DeleteComment(c *gin.Context) {
	item := ownComment(c)
	if item == nil {
		return
	}
	dbResponse(c, config.DB.Model(item).Updates(map[string]interface{}{"content": "", "removed_at": time.Now()}).Error, gin.H{"removed": true})
}

// ListClueEdits godoc
// @Summary Paginated successful edit events with minimal author identity
// @Tags living_clues
// @Param id path int true "Clue ID"
// @Success 200 {array} models.ClueEdit
// @Router /living_clues/{id}/edits [get]
// @Security Bearer
func ListClueEdits(c *gin.Context) {
	clue := accessibleClue(c)
	if clue == nil {
		return
	}
	limit, offset := page(c)
	items := []models.ClueEdit{}
	err := config.DB.Where("living_clue_id = ?", clue.ID).Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&items).Error
	for i := range items {
		items[i].Author = author(items[i].UserID)
		items[i].Description = "Pista editada"
	}
	dbResponse(c, err, items)
}

// GetLinkPreview godoc
// @Summary Fetch cached public-page metadata without changing the author document
// @Tags living_clues
// @Param url query string true "Public HTTP(S) URL"
// @Success 200 {object} models.LinkPreview
// @Router /link_previews [get]
// @Security Bearer
func GetLinkPreview(c *gin.Context) {
	preview, err := services.GetPublicLinkPreview(c.Request.Context(), c.Query("url"))
	if err != nil {
		rieonError(c, http.StatusBadGateway, "Prévia indisponível. O link original continua disponível.")
		return
	}
	c.JSON(200, preview)
}

// ListClueLinkPreviews godoc
// @Summary Cached previews associated with the current clue document, separate from its author content
// @Tags living_clues
// @Param id path int true "Clue ID"
// @Success 200 {array} models.LinkPreview
// @Router /living_clues/{id}/link_previews [get]
// @Security Bearer
func ListClueLinkPreviews(c *gin.Context) {
	clue := accessibleClue(c)
	if clue == nil {
		return
	}
	items := []models.LinkPreview{}
	err := config.DB.Table("clue_preview_references AS r").Select("r.url, p.title, p.description, p.image, p.updated_at").Joins("LEFT JOIN link_previews p ON p.url = r.url").Where("r.living_clue_id = ?", clue.ID).Scan(&items).Error
	// Older clues predate the derived-reference table; the document stays authoritative.
	if err == nil && len(items) == 0 {
		for _, url := range services.QuillPreviewURLs(clue.DescriptionDelta, clue.Description) {
			var preview models.LinkPreview
			config.DB.Where("url = ?", url).First(&preview)
			preview.URL = url
			items = append(items, preview)
		}
	}
	dbResponse(c, err, items)
}

// ListContributions godoc
// @Summary List clues edited by the authenticated account from recorded edit history
// @Tags preferences
// @Produce json
// @Param limit query int false "Page size (maximum 50)"
// @Param offset query int false "Offset"
// @Success 200 {array} map[string]interface{}
// @Router /me/contributions [get]
// @Security Bearer
func ListContributions(c *gin.Context) {
	limit, offset := page(c)
	u := currentUser(c)
	type contribution struct {
		ID               uint      `json:"id"`
		Name             string    `json:"name"`
		LastContribution time.Time `json:"last_contribution"`
	}
	items := []contribution{}
	err := config.DB.Model(&models.LivingClue{}).
		Select("living_clues.id, living_clues.name, MAX(clue_edits.created_at) AS last_contribution").
		Joins("JOIN clue_edits ON clue_edits.living_clue_id = living_clues.id").
		Where("clue_edits.user_id = ?", u.ID).
		Group("living_clues.id, living_clues.name").
		Order("last_contribution DESC, living_clues.id DESC").Limit(limit).Offset(offset).Scan(&items).Error
	dbResponse(c, err, items)
}
