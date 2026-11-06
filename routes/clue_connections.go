package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterClueConnectionRoutes(router *gin.RouterGroup) {
	router.GET("/clue_connections", controllers.FindClueConnections)
	router.GET("/clue_connections/:id", controllers.FindClueConnection)
	router.POST("/clue_connections", controllers.CreateClueConnection)
	router.PUT("/clue_connections/:id", controllers.UpdateClueConnection)
	router.DELETE("/clue_connections/:id", controllers.DeleteClueConnection)
}
