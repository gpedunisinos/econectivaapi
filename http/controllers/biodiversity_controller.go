package controllers

import (
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

// FindNearbyFauna godoc
// @Summary Fauna observada no entorno
// @Description Up to 12 species from a sample of 200 recent iNaturalist observations with licensed photos. Public, possibly approximate coordinates only; no clue is created.
// @Tags biodiversity
// @Produce json
// @Param latitude query number true "Latitude"
// @Param longitude query number true "Longitude"
// @Param radius_m query int false "Radius in meters" Enums(500,1000,5000) default(1000)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.APIMessage
// @Failure 503 {object} models.APIMessage
// @Router /biodiversity/nearby [get]
// @Security Bearer
func FindNearbyFauna(c *gin.Context) { biodiversityResponse(c, true) }

// FindFaunaObservations godoc
// @Summary Public fauna observations in the map viewport
// @Description Up to 100 recent observations; each axis must span at most 3 degrees.
// @Tags biodiversity
// @Produce json
// @Param north query number true "North"
// @Param south query number true "South"
// @Param east query number true "East"
// @Param west query number true "West"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.APIMessage
// @Failure 503 {object} models.APIMessage
// @Router /biodiversity/observations [get]
// @Security Bearer
func FindFaunaObservations(c *gin.Context) { biodiversityResponse(c, false) }

func biodiversityResponse(c *gin.Context, nearby bool) {
	params, err := services.BiodiversityParameters(c.Request.URL.Query(), nearby)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	body, err := services.FetchBiodiversity(c.Request.Context(), params, nearby)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "Fauna temporariamente indisponível. Tente novamente."})
		return
	}
	c.Header("Cache-Control", "private, max-age=60")
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}
