package routes

import (
	"github.com/OdairPianta/unisinos_usp_api/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterEconectivaRoutes(r *gin.RouterGroup) {
	r.GET("/me/preferences", controllers.GetPreferences)
	r.PUT("/me/preferences", controllers.SavePreferences)
	r.GET("/me/locker", controllers.ListLocker)
	r.GET("/me/contributions", controllers.ListContributions)
	r.PUT("/me/locker/:type/:entity", controllers.SaveLocker)
	r.DELETE("/me/locker/:type/:entity", controllers.RemoveLocker)
	r.GET("/living_clues/:id/comments", controllers.ListComments)
	r.POST("/living_clues/:id/comments", controllers.CreateComment)
	r.PUT("/living_clues/:id/comments/:comment", controllers.UpdateComment)
	r.DELETE("/living_clues/:id/comments/:comment", controllers.DeleteComment)
	r.GET("/living_clues/:id/edits", controllers.ListClueEdits)
	r.GET("/living_clues/:id/connection_suggestions", controllers.ListClueSuggestions)
	r.POST("/living_clues/:id/connection_suggestions/refresh", controllers.RefreshClueSuggestions)
	r.POST("/living_clues/:id/connection_suggestions/:suggestion/dismiss", controllers.DismissClueSuggestion)
	r.PUT("/living_clues/:id/aliases/:alias", controllers.ReviewClueAlias)
	r.GET("/living_clues/:id/link_previews", controllers.ListClueLinkPreviews)
	r.GET("/link_previews", controllers.GetLinkPreview)
}
