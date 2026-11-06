package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterStatusRoutes(router *gin.RouterGroup) {
	router.GET("/status", controllers.FindStatus)
}
