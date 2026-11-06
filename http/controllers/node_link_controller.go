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

func FindNodeLinks(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewNodeLinkPolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))

	nodeLinks, err := repository.FindNodeLinks(search, &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, nodeLinks)
}

func FindNodeLink(context *gin.Context) {
	nodeLink, err := repository.FindNodeLinkByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Vínculo entre nós não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewNodeLinkPolicy(&user).View(nodeLink) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	context.JSON(http.StatusOK, nodeLink)
}

func CreateNodeLink(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewNodeLinkPolicy(&user).Create() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.CreateNodeLinkInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	if input.SourceNodeID == input.TargetNodeID {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Não é permitido vincular um nó a ele mesmo!"})
		return
	}

	exists, err := repository.NodeLinkExists(input.SourceNodeID, input.TargetNodeID, 0)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	if exists {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Vínculo entre nós já cadastrado!"})
		return
	}

	nodeLink := models.NodeLink{
		SourceNodeID: input.SourceNodeID,
		TargetNodeID: input.TargetNodeID,
	}

	err = repository.CreateNodeLink(&nodeLink)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	createdNodeLink, err := repository.FindNodeLinkByID(fmt.Sprint(nodeLink.ID))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusOK, nodeLink)
		return
	}

	context.JSON(http.StatusOK, createdNodeLink)
}

func UpdateNodeLink(context *gin.Context) {
	nodeLink, err := repository.FindNodeLinkByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Vínculo entre nós não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewNodeLinkPolicy(&user).Update(nodeLink) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.UpdateNodeLinkInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	if input.SourceNodeID == input.TargetNodeID {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Não é permitido vincular um nó a ele mesmo!"})
		return
	}

	exists, err := repository.NodeLinkExists(input.SourceNodeID, input.TargetNodeID, nodeLink.ID)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	if exists {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Vínculo entre nós já cadastrado!"})
		return
	}

	updatedNodeLink := models.NodeLink{
		SourceNodeID: input.SourceNodeID,
		TargetNodeID: input.TargetNodeID,
	}

	err = repository.UpdateNodeLink(nodeLink, &updatedNodeLink)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	nodeLink.SourceNodeID = updatedNodeLink.SourceNodeID
	nodeLink.TargetNodeID = updatedNodeLink.TargetNodeID

	refreshedNodeLink, err := repository.FindNodeLinkByID(fmt.Sprint(nodeLink.ID))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusOK, nodeLink)
		return
	}

	context.JSON(http.StatusOK, refreshedNodeLink)
}

func DeleteNodeLink(context *gin.Context) {
	nodeLink, err := repository.FindNodeLinkByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Vínculo entre nós não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewNodeLinkPolicy(&user).Delete(nodeLink) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	err = repository.DeleteNodeLink(nodeLink)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, nodeLink)
}
