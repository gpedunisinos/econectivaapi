package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/services"
	"os"

	"github.com/OdairPianta/unisinos_usp_api/http/controllers"

	"github.com/OdairPianta/unisinos_usp_api/http/middlewares"
	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/gin-gonic/gin"
)

func HandleRequest() {
	ginMode := os.Getenv("GIN_MODE")
	if ginMode != "" {
		gin.SetMode(ginMode)
	}
	r := gin.New()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/api/presence"}}), gin.Recovery())
	InitRoutes(r)
	port := "8080"
	r.Run(":" + port)
}

func InitRoutes(r *gin.Engine) *gin.Engine {
	requests.RegisterCustomValidators()
	if platform := services.PlatformID(); platform != "" {
		r.Use(func(c *gin.Context) { c.Header("X-Platform", platform); c.Next() })
	}

	r.Static("/storage", "./storage")
	r.Use(middlewares.JSONLogMiddleware(), middlewares.BlockMiddleware(), middlewares.CorsMiddleware())

	public := r.Group("/api")
	RegisteSwaggerRoutes(public)
	RegisterStatusRoutes(public)
	public.GET("/integrations/cdig", controllers.CDIGIntegrationStatus)
	public.GET("/presence", controllers.OpenPresence)
	RegisterLogRoutes(public)
	RegisterAuthRoutes(public)
	RegisterPoliciesRoutes(public)
	RegisterPublicSensorReadingRoutes(public)

	protected := r.Group("/api")
	protected.Use(middlewares.JwtAuthMiddleware())
	protected.GET("/places/search", controllers.SearchPlaces)
	RegisterFilesRoutes(protected)
	RegisterSampleRoutes(protected)
	RegisterSampleDetailRoutes(protected)
	RegisterUserRoutes(protected)
	RegisterEconectivaRoutes(protected)
	RegisterCategoryRoutes(protected)
	RegisterLivingClueRoutes(protected)
	RegisterEnvironmentalLayerRoutes(protected)
	RegisterConnectionTypeRoutes(protected)
	RegisterClueConnectionRoutes(protected)
	RegisterNodeTypeRoutes(protected)
	RegisterNodeRoutes(protected)
	RegisterNodeLinkRoutes(protected)
	RegisterSensorReadingRoutes(protected)
	RegisterChatTemplateRoutes(protected)
	RegisterChatTemplateFileRoutes(protected)
	RegisterAIMessagesRoutes(protected)
	RegisterMeetingRoutes(protected)
	protected.GET("/me/conversations", controllers.ListParticipantConversations)
	protected.POST("/me/conversations", controllers.OpenParticipantConversation)
	protected.GET("/me/conversations/:id/messages", controllers.GetParticipantMessages)
	protected.POST("/me/conversations/:id/messages", controllers.SendParticipantMessage)
	protected.POST("/me/conversations/:id/read", controllers.ReadParticipantMessages)
	protected.POST("/me/conversations/:id/report", controllers.ReportParticipantConversation)
	protected.POST("/me/conversations/:id/block", controllers.BlockParticipantConversation)
	protected.DELETE("/me/conversations/:id/block", controllers.UnblockParticipantConversation)
	registerIareteMCP(protected)
	return r
}
