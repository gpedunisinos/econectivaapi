package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterLivingClueRoutes(router *gin.RouterGroup) {
	router.GET("/living_clues", controllers.FindLivingClues)
	router.GET("/living_clues/:id", controllers.FindLivingClue)
	router.GET("/living_clues/:id/connections", controllers.FindLivingClueConnections)
	router.POST("/living_clues", controllers.CreateLivingClue)
	router.PUT("/living_clues/:id", controllers.UpdateLivingClue)
	router.POST("/living_clues/:id/environmental_context/reprocess", controllers.ReprocessLivingClueEnvironmentalContext)
	router.DELETE("/living_clues/:id", controllers.DeleteLivingClue)
	router.GET("/living_clues_select", controllers.SelectLivingClues)
	RegisterLivingClueChatRoutes(router)
}
