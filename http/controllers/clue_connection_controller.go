package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/policies"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

func FindClueConnections(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewClueConnectionPolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))

	clueConnections, err := repository.FindClueConnections(search, &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, clueConnections)
}

// @Summary Get the connections of a living clue
// @Description Get incoming and outgoing connections of a living clue, including related clues and connection types
// @Tags living_clues
// @Accept json
// @Produce json
// @Param id path int true "Living clue ID"
// @Success 200 {array} models.ClueConnection
// @Failure 404 {object} models.APIMessage "Living clue not found"
// @Router /living_clues/{id}/connections [get]
// @Security Bearer
func FindLivingClueConnections(context *gin.Context) {
	livingClue, err := repository.FindLivingClueReferenceByID(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Pista viva não encontrada!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewLivingCluePolicy(&user).View(livingClue) ||
		!policies.NewClueConnectionPolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	clueConnections, err := repository.FindClueConnectionsByClueID(livingClue.ID)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, clueConnections)
}

func FindClueConnection(context *gin.Context) {
	clueConnection, err := repository.FindClueConnectionByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Conexão não encontrada!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewClueConnectionPolicy(&user).View(clueConnection) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	context.JSON(http.StatusOK, clueConnection)
}

// CreateClueConnection creates an explicit relation between clues.
// @Summary Create a clue connection
// @Description Omitted strength defaults to1; explicit zero is preserved.
// @Tags clue_connections
// @Accept json
// @Produce json
// @Param connection body requests.CreateClueConnectionInput true "Connection"
// @Success 200 {object} models.ClueConnection
// @Failure 400 {object} models.APIMessage
// @Failure 401 {object} models.APIMessage
// @Router /clue_connections [post]
// @Security Bearer
func CreateClueConnection(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewClueConnectionPolicy(&user).Create() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.CreateClueConnectionInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	exists, err := repository.ClueConnectionExists(input.SourceClueID, input.TargetClueID, input.ConnectionTypeID, 0)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	if exists {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Conexão já cadastrada!"})
		return
	}

	strength := 1
	if input.Strength != nil {
		strength = *input.Strength
	}

	clueConnection := models.ClueConnection{
		SourceClueID:     input.SourceClueID,
		TargetClueID:     input.TargetClueID,
		ConnectionTypeID: input.ConnectionTypeID,
		Strength:         strength,
		Evidence:         input.Evidence,
	}

	err = repository.CreateClueConnection(&clueConnection)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, clueConnection)
}

// UpdateClueConnection updates validated relation fields.
// @Summary Update a clue connection
// @Description Explicit zero strength and empty evidence are saved. Omitted/null strength or evidence preserve existing values.
// @Tags clue_connections
// @Accept json
// @Produce json
// @Param id path int true "Connection ID"
// @Param connection body requests.UpdateClueConnectionInput true "Connection"
// @Success 200 {object} models.ClueConnection
// @Failure 400 {object} models.APIMessage
// @Failure 401 {object} models.APIMessage
// @Failure 404 {object} models.APIMessage
// @Router /clue_connections/{id} [put]
// @Security Bearer
func UpdateClueConnection(context *gin.Context) {
	clueConnection, err := repository.FindClueConnectionByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Conexão não encontrada!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewClueConnectionPolicy(&user).Update(clueConnection) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.UpdateClueConnectionInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	exists, err := repository.ClueConnectionExists(input.SourceClueID, input.TargetClueID, input.ConnectionTypeID, clueConnection.ID)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	if exists {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Conexão já cadastrada!"})
		return
	}

	strength := clueConnection.Strength
	if input.Strength != nil {
		strength = *input.Strength
	}

	evidence := clueConnection.Evidence
	if input.Evidence != nil {
		evidence = *input.Evidence
	}

	updatedClueConnection := models.ClueConnection{
		SourceClueID:     input.SourceClueID,
		TargetClueID:     input.TargetClueID,
		ConnectionTypeID: input.ConnectionTypeID,
		Strength:         strength,
		Evidence:         evidence,
	}

	err = repository.UpdateClueConnection(clueConnection, &updatedClueConnection)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	clueConnection.SourceClueID = updatedClueConnection.SourceClueID
	clueConnection.TargetClueID = updatedClueConnection.TargetClueID
	clueConnection.ConnectionTypeID = updatedClueConnection.ConnectionTypeID
	clueConnection.Strength = updatedClueConnection.Strength
	clueConnection.Evidence = updatedClueConnection.Evidence

	context.JSON(http.StatusOK, clueConnection)
}

func DeleteClueConnection(context *gin.Context) {
	clueConnection, err := repository.FindClueConnectionByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Conexão não encontrada!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewClueConnectionPolicy(&user).Delete(clueConnection) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	err = repository.DeleteClueConnection(clueConnection)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, clueConnection)
}
