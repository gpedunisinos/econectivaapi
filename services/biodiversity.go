package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// BiodiversityParameters accepts only the supported bounded read-only query shapes.
func BiodiversityParameters(values url.Values, nearby bool) (url.Values, error) {
	result := url.Values{}
	keys := []string{"north", "south", "east", "west"}
	if nearby {
		keys = []string{"latitude", "longitude"}
	}
	coordinates := map[string]float64{}
	for _, key := range keys {
		value, err := strconv.ParseFloat(values.Get(key), 64)
		max := 180.0
		if key == "latitude" || key == "north" || key == "south" {
			max = 90
		}
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > max {
			return nil, fmt.Errorf("coordenada inválida: %s", key)
		}
		coordinates[key] = value
		result.Set(key, strconv.FormatFloat(value, 'f', -1, 64))
	}
	if nearby {
		radius := values.Get("radius_m")
		if radius == "" {
			radius = "1000"
		}
		if radius != "500" && radius != "1000" && radius != "5000" {
			return nil, fmt.Errorf("raio deve ser 500, 1000 ou 5000 metros")
		}
		result.Set("radius_m", radius)
	} else if coordinates["north"] <= coordinates["south"] || coordinates["east"] <= coordinates["west"] || coordinates["north"]-coordinates["south"] > 3 || coordinates["east"]-coordinates["west"] > 3 {
		return nil, fmt.Errorf("aproxime o mapa para visualizar a fauna")
	}
	return result, nil
}

func FetchBiodiversity(ctx context.Context, parameters url.Values, nearby bool) ([]byte, error) {
	endpoint := strings.TrimRight(os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL"), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("Geo API não configurada")
	}
	path := "/biodiversity/observations"
	if nearby {
		path = "/biodiversity/nearby"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+path+"?"+parameters.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("ENVIRONMENTAL_CONTEXT_API_TOKEN"))
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Geo API status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024+1))
	if err != nil || len(body) > 2*1024*1024 || !json.Valid(body) {
		return nil, fmt.Errorf("resposta de fauna inválida")
	}
	return body, nil
}
