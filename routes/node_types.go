package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterNodeTypeRoutes(router *gin.RouterGroup) {
	router.GET("/node_types", controllers.FindNodeTypes)
	router.GET("/node_types/:id", controllers.FindNodeType)
	router.POST("/node_types", controllers.CreateNodeType)
	router.PUT("/node_types/:id", controllers.UpdateNodeType)
	router.DELETE("/node_types/:id", controllers.DeleteNodeType)
}
