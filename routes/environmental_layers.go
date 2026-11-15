package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterEnvironmentalLayerRoutes(router *gin.RouterGroup) {
	router.GET("/biodiversity/nearby", controllers.FindNearbyFauna)
	router.GET("/biodiversity/observations", controllers.FindFaunaObservations)
	router.GET("/environmental_layers", controllers.FindEnvironmentalLayers)
	router.GET("/environmental_tiles/:layer/:z/:x/:filename", controllers.FindEnvironmentalTile)
}
