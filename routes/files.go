package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterFilesRoutes(router *gin.RouterGroup) {
	router.GET("/files", controllers.FindFiles)
	router.GET("/files/:id/access", controllers.FileAccess)
	router.POST("/files", controllers.CreateFile)
	router.DELETE("/files/:id", controllers.DeleteFile)
}
