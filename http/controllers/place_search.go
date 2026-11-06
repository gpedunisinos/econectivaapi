package controllers

import (
	"errors"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

var placeSearch = services.NewPlaceSearch()

// SearchPlaces searches worldwide places through a globally throttled, cached provider.
// @Summary Search worldwide places
// @Tags Places
// @Security BearerAuth
// @Param q query string true "Place name (3–200 characters)"
// @Success 200 {array} object
// @Failure 429 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /places/search [get]
func SearchPlaces(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	body, err := placeSearch.Search(c.Request.Context(), c.Query("q"))
	if err != nil {
		if errors.Is(err, services.ErrPlaceBusy) {
			c.Header("Retry-After", "2")
			c.JSON(429, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "Não foi possível buscar lugares. Tente novamente em instantes."})
		return
	}
	c.Data(200, "application/json; charset=utf-8", body)
}
