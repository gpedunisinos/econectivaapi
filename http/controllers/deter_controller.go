package controllers

import (
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

// FindNearbyDeter godoc
// @Summary Avisos DETER Amazônia no entorno da pista
// @Description Snapshot local dos últimos 365 dias. Inclui cobertura, período e data de sincronização. Não equivale à taxa PRODES.
// @Tags environmental
// @Produce json
// @Param latitude query number true "Latitude"
// @Param longitude query number true "Longitude"
// @Param radius_m query int false "Raio em metros" Enums(500,1000,5000) default(1000)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.APIMessage
// @Failure 503 {object} models.APIMessage
// @Router /deter/nearby [get]
// @Security Bearer
func FindNearbyDeter(c *gin.Context) {
	params, err := services.BiodiversityParameters(c.Request.URL.Query(), true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	body, err := services.FetchDeter(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "Avisos DETER temporariamente indisponíveis. Tente novamente."})
		return
	}
	c.Header("Cache-Control", "private, max-age=60")
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}
