package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// FetchDeter forwards only validated coordinates; credentials remain on the backend.
func FetchDeter(ctx context.Context, parameters url.Values) ([]byte, error) {
	endpoint := strings.TrimRight(os.Getenv("ENVIRONMENTAL_CONTEXT_API_URL"), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("Geo API não configurada")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/deter/nearby?"+parameters.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("ENVIRONMENTAL_CONTEXT_API_TOKEN"))
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Geo API status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 512*1024+1))
	if err != nil || len(body) > 512*1024 || !json.Valid(body) {
		return nil, fmt.Errorf("resposta DETER inválida")
	}
	return body, nil
}
