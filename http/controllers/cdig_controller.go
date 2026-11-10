package controllers

import (
	"github.com/OdairPianta/unisinos_usp_api/integrations/cdig"
	"github.com/gin-gonic/gin"
	"net/http"
)

// CDIGIntegrationStatus reports public presentation settings, never credentials.
// @Summary Configuração pública da integração CDIG
// @Tags Integrações
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /integrations/cdig [get]
func CDIGIntegrationStatus(c *gin.Context) {
	cfg, _ := cdig.Load()
	result := gin.H{"enabled": cfg.Enabled}
	if cfg.Enabled {
		result["url"] = cfg.BaseURL
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, result)
}
