package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func BlockMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		path := context.Request.URL.Path
		if strings.HasPrefix(path, "/.git") || strings.HasPrefix(path, "/.env") {
			context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão para acessar este recurso"})
			context.Abort()
			return
		}
		context.Next()
	}
}
