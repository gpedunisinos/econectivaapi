package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterLivingClueChatRoutes(router *gin.RouterGroup) {
	router.GET("/living_clues/:id/chat/messages", controllers.FindLivingClueChatMessages)
	router.POST("/living_clues/:id/chat/messages", controllers.SendLivingClueChatMessage)
	router.DELETE("/living_clues/:id/chat/messages", controllers.DeleteLivingClueChatMessages)
}
