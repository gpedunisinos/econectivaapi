package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterChatTemplateRoutes(router *gin.RouterGroup) {
	router.GET("/chat_templates", controllers.FindChatTemplates)
	router.GET("/chat_templates/:id", controllers.FindChatTemplateByID)
	router.POST("/chat_templates", controllers.CreateChatTemplate)
	router.PUT("/chat_templates/:id", controllers.UpdateChatTemplate)
	router.DELETE("/chat_templates/:id", controllers.DeleteChatTemplate)
	router.GET("/chat_templates/:id/messages", controllers.FindChatMessages)
	router.DELETE("/chat_templates/:id/messages", controllers.DeleteChatMessages)
	router.POST("/chat_templates/:id/messages", controllers.SendChatMessage)
}
