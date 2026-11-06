package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterAIMessagesRoutes(router *gin.RouterGroup) {
	router.GET("/ai/messages", controllers.FindAIMessages)
	router.POST("/ai/messages", controllers.SendAIMessage)
	router.POST("/ai/voice/transcriptions", controllers.TranscribeAIVoice)
	router.POST("/ai/voice/speech", controllers.SpeakAIMessage)
}
