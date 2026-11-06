package routes

import (
	"context"
	"net/http"

	"github.com/OdairPianta/unisinos_usp_api/mcpserver"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
)

// @Summary Consultas MCP da IAraeté
// @Description Streamable HTTP autenticado, com sete ferramentas somente leitura. A identidade é obtida da sessão; nenhum argumento de ferramenta concede permissões.
// @Tags ai_messages
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} models.APIMessage
// @Router /mcp/iarete [post]
// @Security Bearer
func registerIareteMCP(router *gin.RouterGroup) {
	services.SetQuerySessionFactory(func(ctx context.Context, user models.User, pinnedClueID uint) (services.QuerySession, error) {
		return mcpserver.ConnectBackend(ctx, mcpserver.NewDatabaseBackend(user, pinnedClueID))
	})
	router.Any("/mcp/iarete", func(c *gin.Context) {
		user := c.MustGet("user").(models.User)
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
		handler := mcpserver.NewHTTPHandler(mcpserver.NewDatabaseBackend(user, 0))
		handler.ServeHTTP(c.Writer, c.Request)
	})
}
