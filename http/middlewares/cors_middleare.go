package middlewares

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CorsMiddleware() gin.HandlerFunc {
	config := cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "app-version", "app-version-id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
	if raw := strings.TrimSpace(os.Getenv("API_ALLOWED_ORIGINS")); raw != "" {
		config.AllowAllOrigins = false
		for _, origin := range strings.Split(raw, ",") {
			if origin = strings.TrimSpace(origin); origin != "" {
				config.AllowOrigins = append(config.AllowOrigins, origin)
			}
		}
	}
	return cors.New(config)
}
