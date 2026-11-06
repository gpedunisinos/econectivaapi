package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterNodeRoutes(router *gin.RouterGroup) {
	router.GET("/nodes", controllers.FindNodes)
	router.GET("/nodes/map", controllers.FindNodesMap)
	router.GET("/nodes/:id", controllers.FindNode)
	router.POST("/nodes", controllers.CreateNode)
	router.PUT("/nodes/:id", controllers.UpdateNode)
	router.DELETE("/nodes/:id", controllers.DeleteNode)
}
