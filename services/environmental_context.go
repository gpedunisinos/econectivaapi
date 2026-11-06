package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/models"
)

const environmentalContextTimeout = 30 * time.Second

type EnvironmentalLayer struct {
	ID             string                   `json:"id"`
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	DataType       string                   `json:"data_type"`
	TileFormat     string                   `json:"tile_format"`
	TileURL        string                   `json:"tile_url"`
	MapRole        string                   `json:"map_role"`
	Attribution    string                   `json:"attribution"`
	DefaultOpacity float64                  `json:"default_opacity"`
	MinZoom        int                      `json:"min_zoom"`
	MaxZoom        int                      `json:"max_zoom"`
	Available      bool                     `json:"available"`
	Legend         []map[string]interface{} `json:"legend"`
	Source         map[string]interface{}   `json:"source"`
}

func environmentalRequest(method, path string, body *bytes.Reader) (*http.Response, error) {
	endpoint := strings.TrimRight(os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL"), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("ENVIRONMENTAL_CONTEXT_API_URL não configurada")
	}
	var reader = body
	if reader == nil {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, endpoint+path, reader)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("ENVIRONMENTAL_CONTEXT_API_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return (&http.Client{Timeout: environmentalContextTimeout}).Do(req)
}

func FetchEnvironmentalLayers() ([]EnvironmentalLayer, error) {
	response, err := environmentalRequest(http.MethodGet, "/layers", nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("serviço ambiental respondeu com status %d", response.StatusCode)
	}
	var layers []EnvironmentalLayer
	if err := json.NewDecoder(response.Body).Decode(&layers); err != nil {
		return nil, err
	}
	for index := range layers {
		format := layers[index].TileFormat
		if format != "jpg" {
			format = "png"
		}
		layers[index].TileURL = "/api/environmental_tiles/" + layers[index].ID +
			"/{z}/{x}/{y}." + format
	}
	return layers, nil
}

func FetchEnvironmentalTile(layer string, z, x, y int, format string) (*http.Response, error) {
	if format != "png" && format != "jpg" {
		return nil, fmt.Errorf("formato de tile inválido")
	}
	// Preserve compatibility with satellite URLs generated before the catalog
	// started advertising the JPEG extension explicitly.
	if layer == "satellite" {
		format = "jpg"
	}
	path := fmt.Sprintf("/tiles/%s/%d/%d/%d.%s", layer, z, x, y, format)
	return environmentalRequest(http.MethodGet, path, nil)
}

type EnvironmentalContextRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type EnvironmentalContextResult struct {
	Biome             string         `json:"biome"`
	LandCover         string         `json:"land_cover"`
	VegetationPercent *float64       `json:"vegetation_percent"`
	WaterDistanceM    *float64       `json:"water_distance_m"`
	AltitudeM         *float64       `json:"altitude_m"`
	SlopePercent      *float64       `json:"slope_percent"`
	BuiltUpPercent    *float64       `json:"built_up_percent"`
	PopulationDensity *float64       `json:"population_density_km2"`
	Sources           models.JSONMap `json:"sources"`
}

func FetchEnvironmentalContext(latitude, longitude float64) (*EnvironmentalContextResult, error) {
	endpoint := strings.TrimRight(os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL"), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("ENVIRONMENTAL_CONTEXT_API_URL não configurada")
	}

	payload, err := json.Marshal(EnvironmentalContextRequest{Latitude: latitude, Longitude: longitude})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint+"/context", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := os.Getenv("ENVIRONMENTAL_CONTEXT_API_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: environmentalContextTimeout}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("serviço ambiental respondeu com status %d", response.StatusCode)
	}

	var result EnvironmentalContextResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Sources == nil {
		result.Sources = models.JSONMap{}
	}
	return &result, nil
}
