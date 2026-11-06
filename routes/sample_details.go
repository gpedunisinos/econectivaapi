package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterSampleDetailRoutes(router *gin.RouterGroup) {
	router.GET("/sample_details_select", controllers.SelectSamples)
}
