package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterEnvironmentalLayerRoutes(router *gin.RouterGroup) {
	router.GET("/environmental_layers", controllers.FindEnvironmentalLayers)
	router.GET("/environmental_tiles/:layer/:z/:x/:filename", controllers.FindEnvironmentalTile)
}
