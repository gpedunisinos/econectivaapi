package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterNodeLinkRoutes(router *gin.RouterGroup) {
	router.GET("/node_links", controllers.FindNodeLinks)
	router.GET("/node_links/:id", controllers.FindNodeLink)
	router.POST("/node_links", controllers.CreateNodeLink)
	router.PUT("/node_links/:id", controllers.UpdateNodeLink)
	router.DELETE("/node_links/:id", controllers.DeleteNodeLink)
}
