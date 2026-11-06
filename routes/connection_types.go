package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterConnectionTypeRoutes(router *gin.RouterGroup) {
	router.GET("/connection_types", controllers.FindConnectionTypes)
	router.GET("/connection_types/:id", controllers.FindConnectionType)
	router.POST("/connection_types", controllers.CreateConnectionType)
	router.PUT("/connection_types/:id", controllers.UpdateConnectionType)
	router.DELETE("/connection_types/:id", controllers.DeleteConnectionType)
	router.GET("/connection_types_select", controllers.SelectConnectionTypes)
}
