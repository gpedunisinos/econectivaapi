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

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/stretchr/testify/assert"
)

type fakeAIChatCompletionService struct {
	Response string
	Inputs   []services.ChatCompletionInput
}

func (f *fakeAIChatCompletionService) GenerateReply(_ context.Context, input services.ChatCompletionInput) (string, error) {
	f.Inputs = append(f.Inputs, input)
	return f.Response, nil
}

func TestAIMessageSendPersistsUserAndAssistantMessages(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()

	fakeService := &fakeAIChatCompletionService{Response: "Resposta da IA"}
	restoreService := services.SetChatCompletionService(fakeService)
	defer restoreService()

	input := map[string]interface{}{
		"message": "O que falta preencher?",
		"context": "Tela: cadastro de aluno\nEtapa: documentos",
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", "/api/ai/messages", bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var response []models.AIMessage
	_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.Len(t, response, 2)
	assert.Equal(t, "user", response[0].Role)
	assert.Equal(t, "assistant", response[1].Role)
	assert.Equal(t, "Tela: cadastro de aluno\nEtapa: documentos", response[0].Context)
	assert.Contains(t, response[1].Context, "history_window_size=50")
	assert.Contains(t, response[1].Context, "Tela: cadastro de aluno")

	assert.Len(t, fakeService.Inputs, 1)
	assert.Equal(t, "O que falta preencher?", fakeService.Inputs[0].Message)
	assert.NotContains(t, fakeService.Inputs[0].Template.SystemInstruction, "Tela: cadastro de aluno", "Client context remains data, never promoted to system instructions")
	assert.Contains(t, fakeService.Inputs[0].Context, "Tela: cadastro de aluno")
	assert.Equal(t, user.ID, fakeService.Inputs[0].User.ID)

	var storedMessages []models.AIMessage
	err := config.DB.Where("user_id = ?", user.ID).Order("id asc").Find(&storedMessages).Error
	assert.Nil(t, err)
	assert.Len(t, storedMessages, 2)
	assert.Equal(t, "O que falta preencher?", storedMessages[0].Content)
	assert.Equal(t, "Resposta da IA", storedMessages[1].Content)
}

func TestAIMessageSendUsesOnlyLast50Messages(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()

	for index := 1; index <= 60; index++ {
		role := "user"
		if index%2 == 0 {
			role = "assistant"
		}
		err := config.DB.Create(&models.AIMessage{
			UserID:  user.ID,
			Role:    role,
			Content: fmt.Sprintf("Mensagem %d", index),
			Context: "Contexto anterior",
		}).Error
		assert.Nil(t, err)
	}

	fakeService := &fakeAIChatCompletionService{Response: "Nova resposta"}
	restoreService := services.SetChatCompletionService(fakeService)
	defer restoreService()

	input := map[string]interface{}{
		"message": "Continue",
		"context": "Tela atual",
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", "/api/ai/messages", bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())
	assert.Len(t, fakeService.Inputs, 1)
	assert.Len(t, fakeService.Inputs[0].History, 50)
	assert.Equal(t, "Mensagem 11", fakeService.Inputs[0].History[0].Content)
	assert.Equal(t, "Mensagem 60", fakeService.Inputs[0].History[49].Content)
}

func TestFindAIMessagesSupportsPagination(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()

	for index := 1; index <= 4; index++ {
		err := config.DB.Create(&models.AIMessage{
			UserID:  user.ID,
			Role:    "user",
			Content: fmt.Sprintf("Mensagem %d", index),
			Context: "Contexto",
		}).Error
		assert.Nil(t, err)
	}

	request, _ := http.NewRequest("GET", "/api/ai/messages?limit=2&offset=1", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var result []models.AIMessage
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Len(t, result, 2)
	assert.Equal(t, "Mensagem 2", result[0].Content)
	assert.Equal(t, "Mensagem 3", result[1].Content)
}

func TestFindAIMessagesReturnsAuthenticatedUserMessagesOnly(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()
	otherUser, _ := initUser()

	err := config.DB.Create(&models.AIMessage{
		UserID:  user.ID,
		Role:    "user",
		Content: "Mensagem do usuário autenticado",
		Context: "Contexto A",
	}).Error
	assert.Nil(t, err)
	err = config.DB.Create(&models.AIMessage{
		UserID:  otherUser.ID,
		Role:    "user",
		Content: "Mensagem de outro usuário",
		Context: "Contexto B",
	}).Error
	assert.Nil(t, err)

	request, _ := http.NewRequest("GET", "/api/ai/messages", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var result []models.AIMessage
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Len(t, result, 1)
	assert.Equal(t, "Mensagem do usuário autenticado", result[0].Content)
	assert.False(t, strings.Contains(recorder.Body.String(), "outro usuário"))
}
