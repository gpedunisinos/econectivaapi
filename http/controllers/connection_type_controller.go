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
	"gorm.io/gorm"
)

func FindConnectionTypes(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewConnectionTypePolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))

	connectionTypes, err := repository.FindConnectionTypes(search, &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, connectionTypes)
}

func FindConnectionType(context *gin.Context) {
	connectionType, err := repository.FindConnectionTypeByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Tipo de conexão não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewConnectionTypePolicy(&user).View(connectionType) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	context.JSON(http.StatusOK, connectionType)
}

func CreateConnectionType(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewConnectionTypePolicy(&user).Create() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.CreateConnectionTypeInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	_, err := repository.FindConnectionTypeByName(input.Name)
	if err == nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Tipo de conexão já cadastrado!"})
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	isDirectional := true
	if input.IsDirectional != nil {
		isDirectional = *input.IsDirectional
	}

	connectionType := models.ConnectionType{
		Name:          input.Name,
		StrokeStyle:   input.StrokeStyle,
		IsDirectional: isDirectional,
	}

	err = repository.CreateConnectionType(&connectionType)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, connectionType)
}

func UpdateConnectionType(context *gin.Context) {
	connectionType, err := repository.FindConnectionTypeByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Tipo de conexão não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewConnectionTypePolicy(&user).Update(connectionType) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.UpdateConnectionTypeInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	existsConnectionType, err := repository.FindConnectionTypeByName(input.Name)
	if err == nil && existsConnectionType.ID != connectionType.ID {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Tipo de conexão já cadastrado!"})
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	isDirectional := connectionType.IsDirectional
	if input.IsDirectional != nil {
		isDirectional = *input.IsDirectional
	}

	updatedConnectionType := models.ConnectionType{
		Name:          input.Name,
		StrokeStyle:   input.StrokeStyle,
		IsDirectional: isDirectional,
	}

	err = repository.UpdateConnectionType(connectionType, &updatedConnectionType)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	connectionType.Name = updatedConnectionType.Name
	connectionType.StrokeStyle = updatedConnectionType.StrokeStyle
	connectionType.IsDirectional = updatedConnectionType.IsDirectional

	context.JSON(http.StatusOK, connectionType)
}

func DeleteConnectionType(context *gin.Context) {
	connectionType, err := repository.FindConnectionTypeByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Tipo de conexão não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewConnectionTypePolicy(&user).Delete(connectionType) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	err = repository.DeleteConnectionType(connectionType)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, connectionType)
}

func SelectConnectionTypes(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewConnectionTypePolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))

	itemSelects, err := repository.FindConnectionTypeSelects(search, &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, itemSelects)
}
