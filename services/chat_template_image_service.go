package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/models"
)

const (
	chatTemplateImageMaxSide = 1024
	chatTemplateImageQuality = 70
)

type ChatTemplateImageDescriber interface {
	DescribeImage(ctx context.Context, imageBytes []byte, mimeType string, name string) (string, error)
}

type ChatTemplateImageSaver interface {
	Save(imageBytes []byte, name string) (*models.File, error)
}

type GeminiImageDescriber struct {
	Client *http.Client
}

type GryphonImageSaver struct{}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiVisionPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiVisionContent struct {
	Role  string             `json:"role,omitempty"`
	Parts []geminiVisionPart `json:"parts"`
}

type geminiVisionRequest struct {
	Contents []geminiVisionContent `json:"contents"`
}

var chatTemplateImageDescriber ChatTemplateImageDescriber = NewGeminiImageDescriber(http.DefaultClient)
var chatTemplateImageSaver ChatTemplateImageSaver = GryphonImageSaver{}

func NewGeminiImageDescriber(client *http.Client) *GeminiImageDescriber {
	if client == nil {
		client = http.DefaultClient
	}
	return &GeminiImageDescriber{Client: client}
}

func GetChatTemplateImageDescriber() ChatTemplateImageDescriber {
	return chatTemplateImageDescriber
}

func GetChatTemplateImageSaver() ChatTemplateImageSaver {
	return chatTemplateImageSaver
}

func SetChatTemplateImageDescriber(service ChatTemplateImageDescriber) func() {
	previous := chatTemplateImageDescriber
	if service == nil {
		chatTemplateImageDescriber = NewGeminiImageDescriber(http.DefaultClient)
	} else {
		chatTemplateImageDescriber = service
	}
	return func() { chatTemplateImageDescriber = previous }
}

func SetChatTemplateImageSaver(service ChatTemplateImageSaver) func() {
	previous := chatTemplateImageSaver
	if service == nil {
		chatTemplateImageSaver = GryphonImageSaver{}
	} else {
		chatTemplateImageSaver = service
	}
	return func() { chatTemplateImageSaver = previous }
}

func IsImageMimeType(mimeType string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	return strings.HasPrefix(mimeType, "image/")
}

func OptimizeImageToJPEG(body []byte) ([]byte, string, error) {
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	newWidth, newHeight := width, height
	maxSide := max(width, height)
	if maxSide > chatTemplateImageMaxSide {
		scale := float64(chatTemplateImageMaxSide) / float64(maxSide)
		newWidth = max(1, int(math.Round(float64(width)*scale)))
		newHeight = max(1, int(math.Round(float64(height)*scale)))
	}
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	resizeNearest(resized, img)
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, resized, &jpeg.Options{Quality: chatTemplateImageQuality}); err != nil {
		return nil, "", err
	}
	return buffer.Bytes(), "image/jpeg", nil
}

func resizeNearest(dst *image.RGBA, src image.Image) {
	sb := src.Bounds()
	db := dst.Bounds()
	scaleX := float64(sb.Dx()) / float64(db.Dx())
	scaleY := float64(sb.Dy()) / float64(db.Dy())
	for y := db.Min.Y; y < db.Max.Y; y++ {
		sy := sb.Min.Y + int(float64(y-db.Min.Y)*scaleY)
		if sy >= sb.Max.Y {
			sy = sb.Max.Y - 1
		}
		for x := db.Min.X; x < db.Max.X; x++ {
			sx := sb.Min.X + int(float64(x-db.Min.X)*scaleX)
			if sx >= sb.Max.X {
				sx = sb.Max.X - 1
			}
			dst.Set(x, y, src.At(sx, sy))
		}
	}
}

func (GryphonImageSaver) Save(imageBytes []byte, name string) (*models.File, error) {
	ext := "jpg"
	encoded := base64.StdEncoding.EncodeToString(imageBytes)
	publicURL, err := models.SaveGryphonAPI(encoded, ext, name)
	if err != nil {
		return nil, err
	}
	file := &models.File{Extension: ext, PublicURL: publicURL, Name: name}
	return file, nil
}

func (s *GeminiImageDescriber) DescribeImage(ctx context.Context, imageBytes []byte, mimeType string, name string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	apiKey := os.Getenv("GENAI_API_KEY")
	if apiKey == "" {
		return "", errors.New("GENAI_API_KEY não configurada")
	}
	modelName := GeminiModelName()
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", modelName)
	prompt := fmt.Sprintf("Analise esta imagem anexada ao contexto de um chat customizado. Nome do arquivo: %s. Descreva de forma objetiva em português: (1) o que aparece, (2) qualquer texto visível, (3) se parece documento, print, foto, mapa, diagrama ou gráfico, (4) os pontos mais úteis para usar como contexto. Não invente conteúdo não visível.", name)
	payload := geminiVisionRequest{Contents: []geminiVisionContent{{Role: "user", Parts: []geminiVisionPart{{Text: prompt}, {InlineData: &geminiInlineData{MimeType: mimeType, Data: base64.StdEncoding.EncodeToString(imageBytes)}}}}}}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-goog-api-key", apiKey)
	response, err := s.Client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	var result geminiResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", err
	}
	if response.StatusCode >= http.StatusBadRequest {
		return "", ErrGeminiUnavailable
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("Gemini não retornou descrição da imagem")
	}
	return strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text), nil
}
