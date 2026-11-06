package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterPoliciesRoutes(router *gin.RouterGroup) {
	router.GET("/policy/privacy", controllers.FindPolicyPrivacy)
	router.GET("/policy/delete_user_data", controllers.FindPolicyDeleteUserData)
}
