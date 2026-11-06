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

func FindNodeTypes(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewNodeTypePolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))

	nodeTypes, err := repository.FindNodeTypes(search, &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, nodeTypes)
}

func FindNodeType(context *gin.Context) {
	nodeType, err := repository.FindNodeTypeByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Tipo de nó não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewNodeTypePolicy(&user).View(nodeType) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	context.JSON(http.StatusOK, nodeType)
}

func CreateNodeType(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewNodeTypePolicy(&user).Create() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.CreateNodeTypeInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	_, err := repository.FindNodeTypeByName(input.Name)
	if err == nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Tipo de nó já cadastrado!"})
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	nodeType := models.NodeType{
		Name:  input.Name,
		Color: input.Color,
	}

	err = repository.CreateNodeType(&nodeType)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, nodeType)
}

func UpdateNodeType(context *gin.Context) {
	nodeType, err := repository.FindNodeTypeByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Tipo de nó não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewNodeTypePolicy(&user).Update(nodeType) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.UpdateNodeTypeInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	existingNodeType, err := repository.FindNodeTypeByName(input.Name)
	if err == nil && existingNodeType.ID != nodeType.ID {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Tipo de nó já cadastrado!"})
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	updatedNodeType := models.NodeType{
		Name:  input.Name,
		Color: input.Color,
	}

	err = repository.UpdateNodeType(nodeType, &updatedNodeType)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	nodeType.Name = updatedNodeType.Name
	nodeType.Color = updatedNodeType.Color

	context.JSON(http.StatusOK, nodeType)
}

func DeleteNodeType(context *gin.Context) {
	nodeType, err := repository.FindNodeTypeByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Tipo de nó não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewNodeTypePolicy(&user).Delete(nodeType) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	err = repository.DeleteNodeType(nodeType)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, nodeType)
}
