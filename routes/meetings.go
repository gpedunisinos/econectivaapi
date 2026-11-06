package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterMeetingRoutes(router *gin.RouterGroup) {
	router.POST("/me/meetings/access", controllers.CreateMeetingAccess)
}
