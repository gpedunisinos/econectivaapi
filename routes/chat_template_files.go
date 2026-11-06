package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterChatTemplateFileRoutes(router *gin.RouterGroup) {
	router.GET("/chat_template_files", controllers.FindChatTemplateFiles)
	router.POST("/chat_template_files", controllers.CreateChatTemplateFile)
	router.GET("/chat_template_files/:id", controllers.FindChatTemplateFileByID)
	router.DELETE("/chat_template_files/:id", controllers.DeleteChatTemplateFile)
}
