package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(router *gin.RouterGroup) {
	router.POST("/register", controllers.Register)
	router.POST("/login", controllers.Login)
	router.POST("/login/google", controllers.LoginGoogle)
	router.POST("/login/apple", controllers.LoginApple)
	router.POST("/forgot_password", controllers.ForgotPassword)
	router.POST("/recover_password", controllers.RecoverPassword)
}
