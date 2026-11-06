package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/policies"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ClueDiscoveryResponse struct {
	Status      string                  `json:"status"`
	Message     string                  `json:"message,omitempty"`
	Suggestions []models.ClueSuggestion `json:"suggestions"`
	Aliases     []models.ClueAlias      `json:"aliases"`
	Unresolved  interface{}             `json:"unresolved"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

// ListClueSuggestions godoc
// @Summary Suggested connections grounded in explicit names and network evidence
// @Description Lazily schedules background analysis for changed content. Suggestions never create connections. Disallowed, deleted, inactive, self and already connected targets are omitted.
// @Tags living_clues
// @Param id path int true "Clue ID"
// @Success 200 {object} ClueDiscoveryResponse
// @Router /living_clues/{id}/connection_suggestions [get]
// @Security Bearer
func ListClueSuggestions(c *gin.Context) {
	clue := accessibleClue(c)
	if clue == nil {
		return
	}
	result := ClueDiscoveryResponse{Status: "unavailable", Suggestions: []models.ClueSuggestion{}, Aliases: []models.ClueAlias{}, Unresolved: []interface{}{}}
	if err := config.DB.Where("living_clue_id = ? AND status IN ?", clue.ID, []string{"approved", "pending"}).Order("status, name").Find(&result.Aliases).Error; err != nil {
		dbResponse(c, err, nil)
		return
	}
	if !services.DiscoveryEnabled() {
		result.Message = "As sugestões de conexão ainda não estão habilitadas neste ambiente."
		c.JSON(200, result)
		return
	}
	if err := services.QueueClueDiscovery(clue.ID, false); err != nil {
		result.Status = "failed"
		result.Message = "Não foi possível analisar esta pista. O cadastro continua disponível."
		c.JSON(200, result)
		return
	}
	var state models.ClueDiscoveryState
	if err := config.DB.First(&state, "living_clue_id = ?", clue.ID).Error; err != nil {
		dbResponse(c, err, nil)
		return
	}
	result.Status = state.Status
	result.UpdatedAt = state.UpdatedAt
	if state.Status == "ready" {
		query := config.DB.Model(&models.ClueSuggestion{}).Joins("JOIN living_clues target ON target.id = clue_suggestions.target_clue_id AND target.deleted_at IS NULL AND target.is_active = ?", true).
			Where("clue_suggestions.source_clue_id = ? AND clue_suggestions.target_clue_id <> ? AND clue_suggestions.source_hash = ? AND clue_suggestions.dismissed = ?", clue.ID, clue.ID, state.ContentHash, false).
			Where("NOT EXISTS (SELECT 1 FROM clue_connections cc WHERE cc.deleted_at IS NULL AND ((cc.source_clue_id = ? AND cc.target_clue_id = target.id) OR (cc.target_clue_id = ? AND cc.source_clue_id = target.id)))", clue.ID, clue.ID).
			Preload("TargetClue", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name", "category_id", "description", "is_active") }).Preload("TargetClue.Category").Order("clue_suggestions.score DESC, clue_suggestions.target_clue_id").Limit(5)
		if err := query.Find(&result.Suggestions).Error; err != nil {
			dbResponse(c, err, nil)
			return
		}
		permitted := []models.ClueSuggestion{}
		policy := policies.NewLivingCluePolicy(func() *models.User { u := c.MustGet("user").(models.User); return &u }())
		for _, item := range result.Suggestions {
			if item.TargetClue != nil && policy.View(item.TargetClue) {
				permitted = append(permitted, item)
			}
		}
		result.Suggestions = permitted
		if state.Unresolved != nil {
			if entities, ok := state.Unresolved["entities"]; ok && entities != nil {
				result.Unresolved = entities
			}
		}
	}
	if state.Status == "failed" {
		result.Message = "A análise está temporariamente indisponível. Você pode tentar novamente; sua pista está salva."
	}
	c.JSON(http.StatusOK, result)
}

// RefreshClueSuggestions godoc
// @Summary Request background reference analysis again
// @Tags living_clues
// @Param id path int true "Clue ID"
// @Success 202 {object} models.APIMessage
// @Router /living_clues/{id}/connection_suggestions/refresh [post]
// @Security Bearer
func RefreshClueSuggestions(c *gin.Context) {
	clue := accessibleClue(c)
	if clue == nil {
		return
	}
	if !services.DiscoveryEnabled() {
		c.JSON(503, gin.H{"message": "Análise indisponível neste ambiente."})
		return
	}
	if err := services.QueueClueDiscovery(clue.ID, true); err != nil {
		c.JSON(503, gin.H{"message": "Não foi possível agendar a análise."})
		return
	}
	c.JSON(202, gin.H{"message": "Análise agendada."})
}

// DismissClueSuggestion godoc
// @Summary Dismiss a connection suggestion for its current content version
// @Tags living_clues
// @Param id path int true "Clue ID"
// @Param suggestion path int true "Suggestion ID"
// @Success 200 {object} models.APIMessage
// @Router /living_clues/{id}/connection_suggestions/{suggestion}/dismiss [post]
// @Security Bearer
func DismissClueSuggestion(c *gin.Context) {
	clue := accessibleClue(c)
	if clue == nil {
		return
	}
	id, err := strconv.ParseUint(c.Param("suggestion"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"message": "Sugestão inválida."})
		return
	}
	update := config.DB.Model(&models.ClueSuggestion{}).Where("id = ? AND source_clue_id = ?", id, clue.ID).Update("dismissed", true)
	if update.Error != nil {
		dbResponse(c, update.Error, nil)
		return
	}
	if update.RowsAffected == 0 {
		c.JSON(404, gin.H{"message": "Sugestão não encontrada."})
		return
	}
	c.JSON(200, gin.H{"message": "Sugestão dispensada."})
}

// ReviewClueAlias godoc
// @Summary Approve or reject an extracted alternative name
// @Tags living_clues
// @Param id path int true "Clue ID"
// @Param alias path int true "Alias ID"
// @Param input body object true "status: approved or rejected"
// @Success 200 {object} models.APIMessage
// @Router /living_clues/{id}/aliases/{alias} [put]
// @Security Bearer
func ReviewClueAlias(c *gin.Context) {
	clue := accessibleClue(c)
	if clue == nil {
		return
	}
	var input struct {
		Status string `json:"status" binding:"required,oneof=approved rejected"`
	}
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(400, gin.H{"message": "Escolha aprovar ou dispensar o nome alternativo."})
		return
	}
	var alias models.ClueAlias
	if config.DB.First(&alias, "id = ? AND living_clue_id = ? AND status = ?", c.Param("alias"), clue.ID, "pending").Error != nil {
		c.JSON(404, gin.H{"message": "Nome pendente não encontrado."})
		return
	}
	user := c.MustGet("user").(models.User)
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		update := tx.Model(&models.ClueAlias{}).Where("id = ? AND living_clue_id = ? AND status = ?", alias.ID, clue.ID, "pending").Update("status", input.Status)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Create(&models.ClueEdit{LivingClueID: clue.ID, UserID: user.ID, EventType: "updated"}).Error
	})
	if err != nil {
		dbResponse(c, err, nil)
		return
	}
	services.QueueClueDiscovery(clue.ID, false)
	c.JSON(200, gin.H{"message": "Nome alternativo revisado."})
}
