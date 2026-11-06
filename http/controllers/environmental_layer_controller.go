package controllers

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
)

// @Summary List environmental map layers
// @Description Returns available environmental layers, styles, sources and legends
// @Tags environmental layers
// @Produce json
// @Success 200 {array} services.EnvironmentalLayer
// @Router /environmental_layers [get]
func FindEnvironmentalLayers(context *gin.Context) {
	layers, err := services.FetchEnvironmentalLayers()
	if err != nil {
		log.Printf("environmental layers unavailable: %v", err)
		context.JSON(http.StatusOK, []services.EnvironmentalLayer{})
		return
	}
	context.JSON(http.StatusOK, layers)
}

// @Summary Get an environmental map tile
// @Description Proxies an authenticated PNG tile from the Geo API
// @Tags environmental layers
// @Produce image/png,image/jpeg
// @Param layer path string true "Layer identifier"
// @Param z path int true "Zoom"
// @Param x path int true "Tile X"
// @Param y path int true "Tile Y"
// @Success 200 {file} binary
// @Failure 502 {object} models.APIMessage
// @Router /environmental_tiles/{layer}/{z}/{x}/{y}.png [get]
// @Router /environmental_tiles/{layer}/{z}/{x}/{y}.jpg [get]
func FindEnvironmentalTile(context *gin.Context) {
	z, errZ := strconv.Atoi(context.Param("z"))
	x, errX := strconv.Atoi(context.Param("x"))
	yText, format, hasFormat := strings.Cut(context.Param("filename"), ".")
	y, errY := strconv.Atoi(yText)
	if errZ != nil || errX != nil || errY != nil || !hasFormat || (format != "png" && format != "jpg") {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Coordenadas do tile inválidas"})
		return
	}
	response, err := services.FetchEnvironmentalTile(context.Param("layer"), z, x, y, format)
	if err != nil {
		context.JSON(http.StatusBadGateway, gin.H{"message": "Não foi possível carregar o tile ambiental"})
		return
	}
	defer response.Body.Close()
	for _, header := range []string{"Content-Type", "Cache-Control", "ETag"} {
		if value := response.Header.Get(header); value != "" {
			context.Header(header, value)
		}
	}
	context.Status(response.StatusCode)
	_, _ = io.Copy(context.Writer, response.Body)
}
