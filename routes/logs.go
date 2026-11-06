package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterLogRoutes(router *gin.RouterGroup) {
	router.GET("/log/:access_hash", controllers.FindLog)
}
