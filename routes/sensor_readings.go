package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterPublicSensorReadingRoutes(router *gin.RouterGroup) {
	router.POST("/living_clues/:id/sensor_readings", controllers.CreateSensorReading)
}

func RegisterSensorReadingRoutes(router *gin.RouterGroup) {
	router.GET("/sensor_readings", controllers.FindSensorReadings)
	router.GET("/sensor_readings/:id", controllers.FindSensorReading)
	router.DELETE("/sensor_readings/:id", controllers.DeleteSensorReading)
}
