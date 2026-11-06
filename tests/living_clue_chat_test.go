package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/assert"
)

type fakeLivingClueChatService struct {
	Response string
	Inputs   []services.ChatCompletionInput
}

func (f *fakeLivingClueChatService) GenerateReply(_ context.Context, input services.ChatCompletionInput) (string, error) {
	f.Inputs = append(f.Inputs, input)
	return f.Response, nil
}

func TestLivingClueChatSendPersistsMessagesAndBuildsReadableContext(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()
	livingClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	_ = config.DB.Create(&models.SensorReading{LivingClueID: livingClue.ID, ContentType: "application/json", PayloadRaw: `{"temperature":24.5,"humidity":78}`}).Error
	_ = config.DB.Create(&models.SensorReading{LivingClueID: livingClue.ID, ContentType: "text/plain", PayloadRaw: strings.Repeat("abc", 300)}).Error

	fakeService := &fakeLivingClueChatService{Response: "Eu estou bem e percebo umidade alta."}
	restoreService := services.SetChatCompletionService(fakeService)
	defer restoreService()

	input := map[string]interface{}{"message": "Como você está?"}
	jsonInput, _ := json.Marshal(input)
	request, _ := http.NewRequest("POST", fmt.Sprintf("/api/living_clues/%d/chat/messages", livingClue.ID), bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())
	assert.Len(t, fakeService.Inputs, 1)
	assert.Equal(t, "Como você está?", fakeService.Inputs[0].Message)
	assert.Contains(t, fakeService.Inputs[0].Template.SystemInstruction, "Você é a própria pista viva")
	assert.Contains(t, fakeService.Inputs[0].Context, "temperature=24.5")
	assert.Contains(t, fakeService.Inputs[0].Context, "humidity=78")
	assert.Contains(t, fakeService.Inputs[0].Context, "[truncado]")

	var response []models.LivingClueChatMessage
	_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.Len(t, response, 2)
	assert.Equal(t, user.ID, response[0].UserID)
	assert.Equal(t, "user", response[0].Role)
	assert.Equal(t, "assistant", response[1].Role)
}

func TestLivingClueChatContextIncludesCompletedEnvironmentalContext(t *testing.T) {
	vegetation := 14.34
	waterDistance := 126.05
	altitude := 52.49
	slope := 6.54
	builtUp := 68.28
	populationDensity := 7.51
	analyzedAt := time.Date(2026, 9, 25, 15, 4, 56, 0, time.Local)
	livingClue := &models.LivingClue{
		Name:     "Planta Unisinos",
		Latitude: testCoordinate(-29.9), Longitude: testCoordinate(-51.1),
		EnvironmentalContext: &models.EnvironmentalContext{
			Status:           models.EnvironmentalContextStatusCompleted,
			AnalyzedLatitude: -29.9, AnalyzedLongitude: -51.1,
			Biome:             "Mata Atlântica",
			LandCover:         "Área urbanizada",
			VegetationPercent: &vegetation,
			WaterDistanceM:    &waterDistance,
			AltitudeM:         &altitude,
			SlopePercent:      &slope,
			BuiltUpPercent:    &builtUp,
			PopulationDensity: &populationDensity,
			AnalyzedAt:        &analyzedAt,
		},
	}

	contextText := services.BuildLivingClueChatContext(livingClue, nil)

	assert.Contains(t, contextText, "Contexto socioambiental calculado pela API Geo")
	assert.Contains(t, contextText, "Bioma analisado: Mata Atlântica")
	assert.NotContains(t, contextText, "Bioma cadastrado manualmente")
	assert.Contains(t, contextText, "Cobertura predominante: Área urbanizada")
	assert.Contains(t, contextText, "Vegetação no entorno: 14.34%")
	assert.Contains(t, contextText, "Corpo d'água mais próximo: 126.05 m")
	assert.Contains(t, contextText, "Altitude: 52.49 m")
	assert.Contains(t, contextText, "Declividade média: 6.54%")
	assert.Contains(t, contextText, "Ocupação construída: 68.28%")
	assert.Contains(t, contextText, "Densidade populacional: 7.51 hab./km²")
	assert.Contains(t, contextText, "Analisado em: 2026-09-25 15:04:56")
}

func TestLivingClueChatContextIgnoresIncompleteEnvironmentalContext(t *testing.T) {
	livingClue := &models.LivingClue{
		Name: "Pista em análise",
		EnvironmentalContext: &models.EnvironmentalContext{
			Status:    models.EnvironmentalContextStatusProcessing,
			Biome:     "Não deve aparecer",
			LandCover: "Não deve aparecer",
		},
	}

	contextText := services.BuildLivingClueChatContext(livingClue, nil)

	assert.NotContains(t, contextText, "Contexto socioambiental calculado pela API Geo")
	assert.NotContains(t, contextText, "Não deve aparecer")
}

func TestLivingClueChatUsesOnlyLast30MessagesInHistory(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()
	livingClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)

	for i := 1; i <= 40; i++ {
		role := "user"
		if i%2 == 0 {
			role = "assistant"
		}
		err = config.DB.Create(&models.LivingClueChatMessage{LivingClueID: livingClue.ID, UserID: user.ID, Role: role, Content: fmt.Sprintf("Mensagem %d", i)}).Error
		assert.Nil(t, err)
	}

	fakeService := &fakeLivingClueChatService{Response: "Resposta nova"}
	restoreService := services.SetChatCompletionService(fakeService)
	defer restoreService()

	input := map[string]interface{}{"message": "Continue"}
	jsonInput, _ := json.Marshal(input)
	request, _ := http.NewRequest("POST", fmt.Sprintf("/api/living_clues/%d/chat/messages", livingClue.ID), bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())
	assert.Len(t, fakeService.Inputs, 1)
	assert.Len(t, fakeService.Inputs[0].History, 30)
	assert.Equal(t, "Mensagem 11", fakeService.Inputs[0].History[0].Content)
	assert.Equal(t, "Mensagem 40", fakeService.Inputs[0].History[29].Content)
}

func TestLivingClueChatMessagesSupportsGetAndDelete(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()
	livingClue, err := factories.CreateLivingClue()
	assert.Nil(t, err)
	assert.Nil(t, config.DB.Create(&models.LivingClueChatMessage{LivingClueID: livingClue.ID, UserID: user.ID, Role: "user", Content: "Oi"}).Error)
	assert.Nil(t, config.DB.Create(&models.LivingClueChatMessage{LivingClueID: livingClue.ID, UserID: user.ID, Role: "assistant", Content: "Olá"}).Error)

	getRequest, _ := http.NewRequest("GET", fmt.Sprintf("/api/living_clues/%d/chat/messages", livingClue.ID), nil)
	getRequest.Header.Set("Authorization", "Bearer "+token)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	assert.Equal(t, http.StatusOK, getRecorder.Code, "Body: "+getRecorder.Body.String())

	var messages []models.LivingClueChatMessage
	_ = json.Unmarshal(getRecorder.Body.Bytes(), &messages)
	assert.Len(t, messages, 2)

	deleteRequest, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/living_clues/%d/chat/messages", livingClue.ID), nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+token)
	deleteRecorder := httptest.NewRecorder()
	router.ServeHTTP(deleteRecorder, deleteRequest)
	assert.Equal(t, http.StatusOK, deleteRecorder.Code, "Body: "+deleteRecorder.Body.String())

	var count int64
	assert.Nil(t, config.DB.Model(&models.LivingClueChatMessage{}).Where("living_clue_id = ? AND user_id = ?", livingClue.ID, user.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}
