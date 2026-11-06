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

func FindNodes(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewNodePolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	search := context.Query("search")
	limit, _ := strconv.Atoi(context.Query("limit"))
	offset, _ := strconv.Atoi(context.Query("offset"))

	nodes, err := repository.FindNodes(search, &limit, &offset)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, nodes)
}

func FindNode(context *gin.Context) {
	node, err := repository.FindNodeByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Nó não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewNodePolicy(&user).View(node) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	context.JSON(http.StatusOK, node)
}

func CreateNode(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewNodePolicy(&user).Create() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.CreateNodeInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	node := models.Node{
		Name:       input.Name,
		Color:      input.Color,
		PositionX:  input.PositionX,
		PositionY:  input.PositionY,
		NodeTypeID: input.NodeTypeID,
	}

	err := repository.CreateNode(&node)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	createdNode, err := repository.FindNodeByID(fmt.Sprint(node.ID))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusOK, node)
		return
	}

	context.JSON(http.StatusOK, createdNode)
}

func UpdateNode(context *gin.Context) {
	node, err := repository.FindNodeByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Nó não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewNodePolicy(&user).Update(node) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	var input requests.UpdateNodeInput
	if err := context.ShouldBindJSON(&input); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": services.GetErrosMessage(err)})
		return
	}

	updatedNode := models.Node{
		Name:       input.Name,
		Color:      input.Color,
		PositionX:  input.PositionX,
		PositionY:  input.PositionY,
		NodeTypeID: input.NodeTypeID,
	}

	err = repository.UpdateNode(node, &updatedNode)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	node.Name = updatedNode.Name
	node.Color = updatedNode.Color
	node.PositionX = updatedNode.PositionX
	node.PositionY = updatedNode.PositionY
	node.NodeTypeID = updatedNode.NodeTypeID

	refreshedNode, err := repository.FindNodeByID(fmt.Sprint(node.ID))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusOK, node)
		return
	}

	context.JSON(http.StatusOK, refreshedNode)
}

func DeleteNode(context *gin.Context) {
	node, err := repository.FindNodeByID(context.Param("id"))
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusNotFound, gin.H{"message": "Nó não encontrado!"})
		return
	}

	user := context.MustGet("user").(models.User)
	if !policies.NewNodePolicy(&user).Delete(node) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	err = repository.DeleteNode(node)
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, node)
}

func FindNodesMap(context *gin.Context) {
	user := context.MustGet("user").(models.User)
	if !policies.NewNodePolicy(&user).ViewAny() {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return
	}

	nodes, nodeLinks, err := repository.FindNodesMapData()
	if err != nil {
		sentry.CaptureException(err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"links": nodeLinks,
	})
}
