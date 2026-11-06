package controllers

import (
	"errors"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type MeetingAccessInput struct {
	Room string `json:"room" binding:"required"`
}

// CreateMeetingAccess godoc
// @Summary Autorizar entrada em encontro sem segundo login
// @Description Qualquer conta autenticada, em ambos os modos, recebe autorização para uma sala por dez minutos. Identidade vem da sessão. Convites públicos não incluem este token.
// @Tags meetings
// @Accept json
// @Produce json
// @Param input body MeetingAccessInput true "Nome da sala"
// @Success 200 {object} services.MeetingAccess
// @Failure 400 {object} models.APIMessage
// @Failure 401 {object} models.APIMessage
// @Failure 503 {object} models.APIMessage
// @Security Bearer
// @Router /me/meetings/access [post]
func CreateMeetingAccess(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var input MeetingAccessInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Informe um nome de sala válido."})
		return
	}
	access, err := services.IssueMeetingAccess(c.MustGet("user").(models.User), input.Room, time.Now())
	if errors.Is(err, services.ErrMeetingRoom) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Use um nome de sala com até 120 letras, números, espaços ou hífens."})
		return
	}
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "Os encontros estão temporariamente indisponíveis. Tente novamente em instantes."})
		return
	}
	c.JSON(http.StatusOK, access)
}
