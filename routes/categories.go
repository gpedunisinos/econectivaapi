package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterCategoryRoutes(router *gin.RouterGroup) {
	router.GET("/categories", controllers.FindCategories)
	router.GET("/categories/:id", controllers.FindCategory)
	router.POST("/categories", controllers.CreateCategory)
	router.PUT("/categories/:id", controllers.UpdateCategory)
	router.DELETE("/categories/:id", controllers.DeleteCategory)
	router.GET("/categories_select", controllers.SelectCategories)
}
