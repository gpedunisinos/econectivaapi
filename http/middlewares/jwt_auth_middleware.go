package middlewares

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services/token"
	"github.com/gin-gonic/gin"
)

func JwtAuthMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		err := token.TokenValid(context)
		if err != nil {
			context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não está logado"})
			context.Abort()
			return
		}
		var userId, errToken = token.ExtractTokenID(context)
		if errToken != nil {
			context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não está logado"})
			context.Abort()
			return
		}
		var stringId = strconv.FormatUint(uint64(userId), 10)
		var user, errUser = repository.FindUserByID(stringId)
		if errUser != nil {
			context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não está logado"})
			context.Abort()
			return
		}
		context.Set("user", *user)
		if user.UsageMode != nil && *user.UsageMode == "explorer" && context.Request.Method != "GET" && context.Request.Method != "OPTIONS" {
			path := context.FullPath()
			ownAccountDeletion := context.Request.Method == http.MethodDelete && path == "/api/users/:id" && context.Param("id") == stringId
			personal := ownAccountDeletion || strings.HasPrefix(path, "/api/me/") || strings.Contains(path, "/comments") || strings.HasSuffix(path, "/messages") || strings.Contains(path, "update_fcm_token") || path == "/api/link_previews" || path == "/api/mcp/iarete" || path == "/api/ai/voice/transcriptions" || path == "/api/ai/voice/speech"
			if !personal {
				context.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Escolha o modo Inventor para criar ou editar conteúdos."})
				return
			}
		}
		context.Next()
	}
}
