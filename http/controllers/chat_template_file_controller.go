package controllers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/policies"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
)

const maxChatTemplateUploadBytes = 12 * 1024 * 1024

func FindChatTemplateFiles(context *gin.Context) {
	chatTemplateID, err := parseUintParam(context.Query("chat_template_id"))
	if err != nil || chatTemplateID == 0 {
		context.JSON(http.StatusBadRequest, gin.H{"message": "chat_template_id inválido"})
		return
	}

	template, ok := loadOwnedChatTemplateByID(context, chatTemplateID)
	if !ok {
		return
	}

	files, err := repository.FindChatTemplateFilesByChatTemplateID(template.ID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}
	context.JSON(http.StatusOK, files)
}

func FindChatTemplateFileByID(context *gin.Context) {
	fileID, err := parseUintParam(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "ID da fonte inválido"})
		return
	}

	file, err := repository.FindChatTemplateFileByID(fileID)
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Fonte não encontrada"})
		return
	}

	_, ok := loadOwnedChatTemplateByID(context, file.ChatTemplateID)
	if !ok {
		return
	}

	context.JSON(http.StatusOK, file)
}

func CreateChatTemplateFile(context *gin.Context) {
	input, uploadedBytes, err := parseChatTemplateFileCreateInput(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	template, ok := loadOwnedChatTemplateByID(context, input.ChatTemplateID)
	if !ok {
		return
	}

	extractor := services.NewChatTemplateExtractor()
	processed := extractor.ProcessInput(models.ChatTemplateFile{
		Name:             strings.TrimSpace(input.Name),
		MimeType:         strings.TrimSpace(input.MimeType),
		SourceType:       strings.TrimSpace(strings.ToLower(input.Type)),
		SourceURL:        strings.TrimSpace(input.URL),
		OriginalContent:  input.Content,
		Content:          input.Content,
		ExtractedContent: input.Content,
	})
	if len(uploadedBytes) > 0 {
		processed.SourceType = "file"
		processed.SourceURL = ""
		processed.OriginalContent = ""
		result := extractor.ProcessUploadedFile(processed.Name, processed.MimeType, uploadedBytes)
		processed.OriginalContent = result.OriginalContent
		processed.Content = result.ExtractedContent
		processed.ExtractedContent = result.ExtractedContent
		processed.MimeType = firstNonEmpty(result.MimeType, processed.MimeType)
		processed.Status = firstNonEmpty(result.Status, "processed")
		processed.ErrorMessage = result.ErrorMessage
	}

	if processed.Status == "failed" {
		context.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("falha ao processar fonte: %s", processed.ErrorMessage)})
		return
	}

	processed.ChatTemplateID = template.ID
	if err := repository.CreateChatTemplateFile(&processed); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, processed)
}

func DeleteChatTemplateFile(context *gin.Context) {
	fileID, err := parseUintParam(context.Param("id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "ID da fonte inválido"})
		return
	}

	file, err := repository.FindChatTemplateFileByID(fileID)
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Fonte não encontrada"})
		return
	}

	_, ok := loadOwnedChatTemplateByID(context, file.ChatTemplateID)
	if !ok {
		return
	}

	if err := repository.DeleteChatTemplateFile(file); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Erro: %s", err.Error())})
		return
	}

	context.JSON(http.StatusOK, file)
}

func loadOwnedChatTemplateByID(context *gin.Context, chatTemplateID uint) (*models.ChatTemplate, bool) {
	user := context.MustGet("user").(models.User)
	template, err := repository.FindChatTemplateByIDAndUserID(strconv.FormatUint(uint64(chatTemplateID), 10), user.ID)
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"message": "Template de chat não encontrado"})
		return nil, false
	}

	if !policies.NewChatTemplatePolicy(&user).Update(template) {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Você não tem permissão!"})
		return nil, false
	}

	return template, true
}

func parseUintParam(value string) (uint, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}

func parseChatTemplateFileCreateInput(context *gin.Context) (requests.CreateChatTemplateFileInput, []byte, error) {
	if strings.HasPrefix(context.GetHeader("Content-Type"), "multipart/form-data") {
		return parseChatTemplateFileCreateMultipartInput(context)
	}
	var input requests.CreateChatTemplateFileInput
	if err := context.ShouldBindJSON(&input); err != nil {
		return requests.CreateChatTemplateFileInput{}, nil, fmt.Errorf(services.GetErrosMessage(err))
	}
	return input, nil, nil
}

func parseChatTemplateFileCreateMultipartInput(context *gin.Context) (requests.CreateChatTemplateFileInput, []byte, error) {
	templateIDValue := strings.TrimSpace(context.PostForm("chat_template_id"))
	if templateIDValue == "" {
		return requests.CreateChatTemplateFileInput{}, nil, fmt.Errorf("chat_template_id é obrigatório")
	}
	templateID, err := parseUintParam(templateIDValue)
	if err != nil || templateID == 0 {
		return requests.CreateChatTemplateFileInput{}, nil, fmt.Errorf("chat_template_id inválido")
	}

	input := requests.CreateChatTemplateFileInput{
		ChatTemplateID: templateID,
		Name:           strings.TrimSpace(context.PostForm("name")),
		MimeType:       strings.TrimSpace(context.PostForm("mime_type")),
		Type:           strings.TrimSpace(strings.ToLower(context.PostForm("type"))),
		URL:            strings.TrimSpace(context.PostForm("url")),
		Content:        context.PostForm("content"),
	}

	upload, err := context.FormFile("file")
	if err == nil {
		bytes, uploadErr := readMultipartFileWithLimit(upload, maxChatTemplateUploadBytes)
		if uploadErr != nil {
			return requests.CreateChatTemplateFileInput{}, nil, uploadErr
		}
		if input.Name == "" {
			input.Name = upload.Filename
		}
		if input.Type == "" {
			input.Type = "file"
		}
		if input.MimeType == "" {
			input.MimeType = upload.Header.Get("Content-Type")
		}
		if input.Name == "" {
			input.Name = "arquivo"
		}
		return input, bytes, nil
	}

	if input.Name == "" {
		return requests.CreateChatTemplateFileInput{}, nil, fmt.Errorf("name é obrigatório")
	}
	if input.Type == "" {
		input.Type = "text"
	}
	return input, nil, nil
}

func readMultipartFileWithLimit(upload *multipart.FileHeader, limit int64) ([]byte, error) {
	file, err := upload.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	limited := io.LimitReader(file, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("arquivo excede o limite de %d bytes", limit)
	}
	return data, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
