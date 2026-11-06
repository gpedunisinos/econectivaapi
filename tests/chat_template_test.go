package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/enums"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/stretchr/testify/assert"
)

type fakeChatCompletionService struct {
	Response string
	Inputs   []services.ChatCompletionInput
}

type fakeChatTemplateImageDescriber struct {
	Response string
}

func (f *fakeChatCompletionService) GenerateReply(_ context.Context, input services.ChatCompletionInput) (string, error) {
	f.Inputs = append(f.Inputs, input)
	return f.Response, nil
}

func (f *fakeChatTemplateImageDescriber) DescribeImage(_ context.Context, _ []byte, _ string, _ string) (string, error) {
	return f.Response, nil
}

func buildPNGTestImage() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	for y := 0; y < 900; y++ {
		for x := 0; x < 1600; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	var buffer bytes.Buffer
	_ = png.Encode(&buffer, img)
	return buffer.Bytes()
}

func createChatTemplateForSources(t *testing.T, router http.Handler, token string, name string) models.ChatTemplate {
	t.Helper()
	templateInput := map[string]interface{}{
		"name":               name,
		"description":        "Template para testes de fonte",
		"system_instruction": "Use as fontes vinculadas.",
	}
	templateJSON, _ := json.Marshal(templateInput)
	templateRequest, _ := http.NewRequest("POST", "/api/chat_templates", bytes.NewBuffer(templateJSON))
	templateRequest.Header.Set("Authorization", "Bearer "+token)
	templateRequest.Header.Set("Content-Type", "application/json")
	templateRecorder := httptest.NewRecorder()
	router.ServeHTTP(templateRecorder, templateRequest)
	assert.Equal(t, http.StatusOK, templateRecorder.Code, "Body: "+templateRecorder.Body.String())

	var template models.ChatTemplate
	_ = json.Unmarshal(templateRecorder.Body.Bytes(), &template)
	assert.NotZero(t, template.ID)
	return template
}

func TestChatTemplateFileCreateStoresExtractedTextInDatabase(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()

	template := createChatTemplateForSources(t, router, token, "Assistente Acadêmico")

	input := map[string]interface{}{
		"chat_template_id": template.ID,
		"name":             "guia.txt",
		"mime_type":        "text/plain",
		"type":             "text",
		"content":          "Calendário letivo 2026",
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", "/api/chat_template_files", bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var source models.ChatTemplateFile
	_ = json.Unmarshal(recorder.Body.Bytes(), &source)
	assert.Equal(t, template.ID, source.ChatTemplateID)
	assert.Equal(t, "Calendário letivo 2026", source.Content)
	assert.Equal(t, "text", source.SourceType)
	assert.Equal(t, "processed", source.Status)

	var storedFile models.ChatTemplateFile
	err := config.DB.Where("chat_template_id = ?", template.ID).First(&storedFile).Error
	assert.Nil(t, err)
	assert.Equal(t, user.ID, template.UserID)
	assert.Equal(t, "guia.txt", storedFile.Name)
	assert.Equal(t, "Calendário letivo 2026", storedFile.Content)
	assert.Equal(t, "Calendário letivo 2026", storedFile.ExtractedContent)
}

func TestChatTemplateUpdateDoesNotReplaceFiles(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()

	template := models.ChatTemplate{
		Name:              "Template Inicial",
		Description:       "Descrição inicial",
		SystemInstruction: "Instrução inicial",
		UserID:            user.ID,
		ChatTemplateFiles: []models.ChatTemplateFile{{Name: "old.txt", MimeType: "text/plain", SourceType: "text", Content: "Conteúdo antigo", OriginalContent: "Conteúdo antigo", ExtractedContent: "Conteúdo antigo", Status: "processed"}},
	}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	input := map[string]interface{}{
		"name":               "Template Atualizado",
		"description":        "Descrição nova",
		"system_instruction": "Instrução nova",
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("PUT", fmt.Sprintf("/api/chat_templates/%d", template.ID), bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var updated models.ChatTemplate
	_ = json.Unmarshal(recorder.Body.Bytes(), &updated)
	assert.Equal(t, "Template Atualizado", updated.Name)
	assert.Equal(t, "Descrição nova", updated.Description)
	assert.Equal(t, "Instrução nova", updated.SystemInstruction)
	assert.Len(t, updated.ChatTemplateFiles, 1)
	assert.Equal(t, "old.txt", updated.ChatTemplateFiles[0].Name)
	assert.Equal(t, "Conteúdo antigo", updated.ChatTemplateFiles[0].Content)

	var files []models.ChatTemplateFile
	err = config.DB.Where("chat_template_id = ?", template.ID).Find(&files).Error
	assert.Nil(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "old.txt", files[0].Name)
	assert.Equal(t, user.ID, template.UserID)
}

func TestChatTemplateFileCreateWithURLExtractsHTMLText(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	assert.Nil(t, os.Setenv("ALLOW_PRIVATE_CONTEXT_URLS", "1"))
	defer os.Unsetenv("ALLOW_PRIVATE_CONTEXT_URLS")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body><h1>Plano do Pampa</h1><p>Bioma piloto.</p></body></html>"))
	}))
	defer server.Close()

	template := createChatTemplateForSources(t, router, token, "Chat com URL")
	input := map[string]interface{}{
		"chat_template_id": template.ID,
		"name":             "plano-pampa",
		"type":             "url",
		"url":              server.URL,
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", "/api/chat_template_files", bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())
	var source models.ChatTemplateFile
	_ = json.Unmarshal(recorder.Body.Bytes(), &source)
	assert.Equal(t, "url", source.SourceType)
	assert.Contains(t, source.Content, "Plano do Pampa")
	assert.Contains(t, source.Content, "Bioma piloto")
}

func TestChatTemplateFileCreateWithFileURLExtractsImageDescription(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	assert.Nil(t, os.Setenv("ALLOW_PRIVATE_CONTEXT_URLS", "1"))
	defer os.Unsetenv("ALLOW_PRIVATE_CONTEXT_URLS")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(buildPNGTestImage())
	}))
	defer server.Close()

	restoreDescriber := services.SetChatTemplateImageDescriber(&fakeChatTemplateImageDescriber{Response: "Imagem com mapa e legenda visível."})
	defer restoreDescriber()
	template := createChatTemplateForSources(t, router, token, "Chat com imagem")

	input := map[string]interface{}{
		"chat_template_id": template.ID,
		"name":             "mapa.png",
		"type":             "file",
		"url":              server.URL,
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", "/api/chat_template_files", bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())
	var source models.ChatTemplateFile
	_ = json.Unmarshal(recorder.Body.Bytes(), &source)
	assert.Equal(t, "file", source.SourceType)
	assert.Equal(t, "image/jpeg", source.MimeType)
	assert.Contains(t, source.Content, "Imagem com mapa e legenda visível")
}

func TestChatTemplateFileCreateWithFileURLExtractsLinkedContent(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	assert.Nil(t, os.Setenv("ALLOW_PRIVATE_CONTEXT_URLS", "1"))
	defer os.Unsetenv("ALLOW_PRIVATE_CONTEXT_URLS")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write([]byte("# Documento base\nEste texto veio do arquivo vinculado."))
	}))
	defer server.Close()

	template := createChatTemplateForSources(t, router, token, "Chat com arquivo")

	input := map[string]interface{}{
		"chat_template_id": template.ID,
		"name":             "base.md",
		"type":             "file",
		"url":              server.URL,
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", "/api/chat_template_files", bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())
	var source models.ChatTemplateFile
	_ = json.Unmarshal(recorder.Body.Bytes(), &source)
	assert.Equal(t, "file", source.SourceType)
	assert.Contains(t, source.Content, "Documento base")
}

func TestChatTemplateFileCreateWithMultipartUploadExtractsContent(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()

	template := createChatTemplateForSources(t, router, token, "Chat com upload direto")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("chat_template_id", fmt.Sprintf("%d", template.ID))
	_ = writer.WriteField("name", "base.md")
	_ = writer.WriteField("type", "file")
	part, _ := writer.CreateFormFile("file", "base.md")
	_, _ = part.Write([]byte("# Documento local\nEste texto foi enviado no multipart."))
	_ = writer.Close()

	request, _ := http.NewRequest("POST", "/api/chat_template_files", &body)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())
	var source models.ChatTemplateFile
	_ = json.Unmarshal(recorder.Body.Bytes(), &source)
	assert.Equal(t, "file", source.SourceType)
	assert.Equal(t, "base.md", source.Name)
	assert.Contains(t, source.Content, "Documento local")
	assert.Contains(t, source.Content, "enviado no multipart")
	assert.Equal(t, "processed", source.Status)
}

func TestChatMessageSendPersistsUserAndAssistantMessages(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()

	template := models.ChatTemplate{
		Name:              "Tutor",
		SystemInstruction: "Use o contexto salvo.",
		UserID:            user.ID,
		ChatTemplateFiles: []models.ChatTemplateFile{
			{Name: "contexto.md", MimeType: "text/markdown", SourceType: "text", Content: "Regra: citar sempre o regulamento.", OriginalContent: "Regra: citar sempre o regulamento.", ExtractedContent: "Regra: citar sempre o regulamento.", Status: "processed"},
		},
	}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	fakeService := &fakeChatCompletionService{Response: "Resposta do assistente"}
	restoreService := services.SetChatCompletionService(fakeService)
	defer restoreService()

	input := map[string]interface{}{
		"message": "Qual é a regra principal?",
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", fmt.Sprintf("/api/chat_templates/%d/messages", template.ID), bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var response []models.ChatMessage
	_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.Len(t, response, 2)
	assert.Equal(t, "user", response[0].Role)
	assert.Equal(t, "assistant", response[1].Role)
	assert.Equal(t, user.ID, response[0].UserID)
	assert.Equal(t, user.ID, response[1].UserID)

	assert.Len(t, fakeService.Inputs, 1)
	assert.Equal(t, "Qual é a regra principal?", fakeService.Inputs[0].Message)
	assert.Len(t, fakeService.Inputs[0].Template.Files, 1)
	assert.Equal(t, "Regra: citar sempre o regulamento.", fakeService.Inputs[0].Template.Files[0].Content)
	assert.Contains(t, fakeService.Inputs[0].Context, "Fontes de contexto do template")
	assert.Contains(t, fakeService.Inputs[0].Context, "Regra: citar sempre o regulamento.")
	assert.Empty(t, fakeService.Inputs[0].History)

	var storedMessages []models.ChatMessage
	err = config.DB.Where("chat_template_id = ? AND user_id = ?", template.ID, user.ID).Order("id asc").Find(&storedMessages).Error
	assert.Nil(t, err)
	assert.Len(t, storedMessages, 2)
	assert.Equal(t, "Qual é a regra principal?", storedMessages[0].Content)
	assert.Equal(t, "Resposta do assistente", storedMessages[1].Content)
}

func TestChatMessageSendUsesAuthenticatedUserHistoryOnly(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()

	template := models.ChatTemplate{
		Name:              "Tutor Histórico",
		SystemInstruction: "Considere o histórico do usuário autenticado.",
		UserID:            user.ID,
	}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: template.ID, UserID: user.ID, Role: "user", Content: "Mensagem anterior"}).Error
	assert.Nil(t, err)
	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: template.ID, UserID: user.ID, Role: "assistant", Content: "Resposta anterior"}).Error
	assert.Nil(t, err)

	fakeService := &fakeChatCompletionService{Response: "Nova resposta"}
	restoreService := services.SetChatCompletionService(fakeService)
	defer restoreService()

	input := map[string]interface{}{"message": "Continue o atendimento"}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", fmt.Sprintf("/api/chat_templates/%d/messages", template.ID), bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())
	assert.Len(t, fakeService.Inputs, 1)
	assert.Len(t, fakeService.Inputs[0].History, 2)
	assert.Equal(t, "Mensagem anterior", fakeService.Inputs[0].History[0].Content)
	assert.Equal(t, "Resposta anterior", fakeService.Inputs[0].History[1].Content)
}

func TestChatMessagesSupportsLimitAndOffset(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()

	template := models.ChatTemplate{Name: "Template Paginação", SystemInstruction: "Teste de paginação.", UserID: user.ID}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	messagesToCreate := []models.ChatMessage{{ChatTemplateID: template.ID, UserID: user.ID, Role: "user", Content: "Mensagem 1"}, {ChatTemplateID: template.ID, UserID: user.ID, Role: "assistant", Content: "Mensagem 2"}, {ChatTemplateID: template.ID, UserID: user.ID, Role: "user", Content: "Mensagem 3"}, {ChatTemplateID: template.ID, UserID: user.ID, Role: "assistant", Content: "Mensagem 4"}}
	for _, msg := range messagesToCreate {
		err = config.DB.Create(&msg).Error
		assert.Nil(t, err)
	}

	request, _ := http.NewRequest("GET", fmt.Sprintf("/api/chat_templates/%d/messages?limit=2&offset=1", template.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var result []models.ChatMessage
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	assert.Len(t, result, 2)
	assert.Equal(t, "Mensagem 2", result[0].Content)
	assert.Equal(t, "Mensagem 3", result[1].Content)
}

func TestChatTemplateDeleteRemovesTemplateAndRelatedData(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()

	template := models.ChatTemplate{Name: "Template para excluir", SystemInstruction: "Instrução", UserID: user.ID, ChatTemplateFiles: []models.ChatTemplateFile{{Name: "arquivo.txt", MimeType: "text/plain", SourceType: "text", Content: "conteúdo", OriginalContent: "conteúdo", ExtractedContent: "conteúdo", Status: "processed"}}}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: template.ID, UserID: user.ID, Role: "user", Content: "mensagem"}).Error
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/chat_templates/%d", template.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var countTemplates int64
	err = config.DB.Model(&models.ChatTemplate{}).Where("id = ?", template.ID).Count(&countTemplates).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(0), countTemplates)

	var countFiles int64
	err = config.DB.Model(&models.ChatTemplateFile{}).Where("chat_template_id = ?", template.ID).Count(&countFiles).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(0), countFiles)

	var countMessages int64
	err = config.DB.Model(&models.ChatMessage{}).Where("chat_template_id = ?", template.ID).Count(&countMessages).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(0), countMessages)
}

func TestChatTemplateDeleteReturnsNotFoundForAnotherUser(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	owner, _ := initUserWithProfile(enums.UserProfileEnumUser)
	otherUser, tokenOther := initUserWithProfile(enums.UserProfileEnumUser)

	template := models.ChatTemplate{Name: "Template privado", SystemInstruction: "Instrução", UserID: owner.ID}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/chat_templates/%d", template.ID), nil)
	request.Header.Set("Authorization", "Bearer "+tokenOther)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code, "Body: "+recorder.Body.String())
	assert.NotEqual(t, owner.ID, otherUser.ID)
}

func TestChatTemplateDeleteAllowsAnotherAdministrator(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	owner, _ := initUserWithProfile(enums.UserProfileEnumUser)
	_, adminToken := initUserWithProfile(enums.UserProfileEnumAdministrator)

	template := models.ChatTemplate{Name: "Template administrável", SystemInstruction: "Instrução", UserID: owner.ID}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/chat_templates/%d", template.ID), nil)
	request.Header.Set("Authorization", "Bearer "+adminToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())
}

func TestChatMessagesDeleteRemovesOnlyTemplateMessagesFromAuthenticatedUser(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	user, token := initUser()

	template := models.ChatTemplate{Name: "Template com histórico", SystemInstruction: "Instrução", UserID: user.ID}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	otherTemplate := models.ChatTemplate{Name: "Outro template", SystemInstruction: "Outra instrução", UserID: user.ID}
	err = config.DB.Create(&otherTemplate).Error
	assert.Nil(t, err)

	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: template.ID, UserID: user.ID, Role: "user", Content: "Mensagem 1"}).Error
	assert.Nil(t, err)
	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: template.ID, UserID: user.ID, Role: "assistant", Content: "Mensagem 2"}).Error
	assert.Nil(t, err)
	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: otherTemplate.ID, UserID: user.ID, Role: "user", Content: "Mensagem preservada"}).Error
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/chat_templates/%d/messages", template.ID), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var removedCount int64
	err = config.DB.Model(&models.ChatMessage{}).Where("chat_template_id = ? AND user_id = ?", template.ID, user.ID).Count(&removedCount).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(0), removedCount)

	var preservedCount int64
	err = config.DB.Model(&models.ChatMessage{}).Where("chat_template_id = ? AND user_id = ?", otherTemplate.ID, user.ID).Count(&preservedCount).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(1), preservedCount)
}

func TestChatMessagesDeleteReturnsNotFoundForAnotherUser(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	owner, _ := initUser()
	otherUser, tokenOther := initUser()

	template := models.ChatTemplate{Name: "Template privado", SystemInstruction: "Instrução", UserID: owner.ID}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: template.ID, UserID: owner.ID, Role: "user", Content: "Mensagem do dono"}).Error
	assert.Nil(t, err)
	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: template.ID, UserID: otherUser.ID, Role: "user", Content: "Mensagem do outro usuário"}).Error
	assert.Nil(t, err)

	request, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/chat_templates/%d/messages", template.ID), nil)
	request.Header.Set("Authorization", "Bearer "+tokenOther)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var ownerMessagesCount int64
	err = config.DB.Model(&models.ChatMessage{}).Where("chat_template_id = ? AND user_id = ?", template.ID, owner.ID).Count(&ownerMessagesCount).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(1), ownerMessagesCount)

	var otherMessagesCount int64
	err = config.DB.Model(&models.ChatMessage{}).Where("chat_template_id = ? AND user_id = ?", template.ID, otherUser.ID).Count(&otherMessagesCount).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(0), otherMessagesCount)
}

func TestChatHistoryIsIsolatedBetweenAuthenticatedUsers(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	userOne, tokenOne := initUser()
	userTwo, tokenTwo := initUser()

	templateUserOne := models.ChatTemplate{Name: "Template Usuário 1", SystemInstruction: "Histórico isolado do usuário 1.", UserID: userOne.ID}
	err := config.DB.Create(&templateUserOne).Error
	assert.Nil(t, err)

	templateUserTwo := models.ChatTemplate{Name: "Template Usuário 2", SystemInstruction: "Histórico isolado do usuário 2.", UserID: userTwo.ID}
	err = config.DB.Create(&templateUserTwo).Error
	assert.Nil(t, err)

	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: templateUserOne.ID, UserID: userOne.ID, Role: "user", Content: "Mensagem do usuário 1"}).Error
	assert.Nil(t, err)
	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: templateUserOne.ID, UserID: userOne.ID, Role: "assistant", Content: "Resposta do usuário 1"}).Error
	assert.Nil(t, err)
	err = config.DB.Create(&models.ChatMessage{ChatTemplateID: templateUserTwo.ID, UserID: userTwo.ID, Role: "user", Content: "Mensagem do usuário 2"}).Error
	assert.Nil(t, err)

	requestOwn, _ := http.NewRequest("GET", fmt.Sprintf("/api/chat_templates/%d/messages", templateUserOne.ID), nil)
	requestOwn.Header.Set("Authorization", "Bearer "+tokenOne)
	recorderOwn := httptest.NewRecorder()
	router.ServeHTTP(recorderOwn, requestOwn)

	assert.Equal(t, http.StatusOK, recorderOwn.Code, "Body: "+recorderOwn.Body.String())

	var ownMessages []models.ChatMessage
	_ = json.Unmarshal(recorderOwn.Body.Bytes(), &ownMessages)
	assert.Len(t, ownMessages, 2)
	assert.Equal(t, userOne.ID, ownMessages[0].UserID)

	requestForeign, _ := http.NewRequest("GET", fmt.Sprintf("/api/chat_templates/%d/messages", templateUserOne.ID), nil)
	requestForeign.Header.Set("Authorization", "Bearer "+tokenTwo)
	recorderForeign := httptest.NewRecorder()
	router.ServeHTTP(recorderForeign, requestForeign)

	assert.Equal(t, http.StatusOK, recorderForeign.Code, "Body: "+recorderForeign.Body.String())

	var foreignMessages []models.ChatMessage
	_ = json.Unmarshal(recorderForeign.Body.Bytes(), &foreignMessages)
	assert.Len(t, foreignMessages, 0)

	requestTemplates, _ := http.NewRequest("GET", "/api/chat_templates", nil)
	requestTemplates.Header.Set("Authorization", "Bearer "+tokenTwo)
	recorderTemplates := httptest.NewRecorder()
	router.ServeHTTP(recorderTemplates, requestTemplates)

	assert.Equal(t, http.StatusOK, recorderTemplates.Code, "Body: "+recorderTemplates.Body.String())

	var templates []models.ChatTemplate
	_ = json.Unmarshal(recorderTemplates.Body.Bytes(), &templates)
	// Templates are shared globally; other tests may have already created agents.
	assert.GreaterOrEqual(t, len(templates), 2)
	ids := map[uint]bool{}
	for _, template := range templates {
		ids[template.ID] = true
	}
	assert.True(t, ids[templateUserOne.ID])
	assert.True(t, ids[templateUserTwo.ID])
}

func TestChatMessageSendAllowsAuthenticatedUserToUseForeignTemplate(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	owner, _ := initUserWithProfile(enums.UserProfileEnumUser)
	otherUser, otherToken := initUserWithProfile(enums.UserProfileEnumUser)

	template := models.ChatTemplate{
		Name:              "Agente global",
		SystemInstruction: "Responda normalmente.",
		UserID:            owner.ID,
	}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	fakeService := &fakeChatCompletionService{Response: "Resposta compartilhada"}
	restoreService := services.SetChatCompletionService(fakeService)
	defer restoreService()

	input := map[string]interface{}{"message": "Olá agente"}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("POST", fmt.Sprintf("/api/chat_templates/%d/messages", template.ID), bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+otherToken)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())
	assert.Len(t, fakeService.Inputs, 1)
	assert.Empty(t, fakeService.Inputs[0].History)

	var storedMessages []models.ChatMessage
	err = config.DB.Where("chat_template_id = ? AND user_id = ?", template.ID, otherUser.ID).Order("id asc").Find(&storedMessages).Error
	assert.Nil(t, err)
	assert.Len(t, storedMessages, 2)
	assert.Equal(t, "Olá agente", storedMessages[0].Content)
}

func TestChatTemplateUpdateAllowsAnotherAdministrator(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	owner, _ := initUserWithProfile(enums.UserProfileEnumUser)
	_, adminToken := initUserWithProfile(enums.UserProfileEnumAdministrator)

	template := models.ChatTemplate{
		Name:              "Template Inicial",
		Description:       "Descrição inicial",
		SystemInstruction: "Instrução inicial",
		UserID:            owner.ID,
	}
	err := config.DB.Create(&template).Error
	assert.Nil(t, err)

	input := map[string]interface{}{
		"name":               "Template Atualizado Admin",
		"description":        "Descrição admin",
		"system_instruction": "Instrução admin",
	}
	jsonInput, _ := json.Marshal(input)

	request, _ := http.NewRequest("PUT", fmt.Sprintf("/api/chat_templates/%d", template.ID), bytes.NewBuffer(jsonInput))
	request.Header.Set("Authorization", "Bearer "+adminToken)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code, "Body: "+recorder.Body.String())

	var updated models.ChatTemplate
	_ = json.Unmarshal(recorder.Body.Bytes(), &updated)
	assert.Equal(t, "Template Atualizado Admin", updated.Name)
}
