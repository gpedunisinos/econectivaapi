package services

import (
	"fmt"
	"strings"

	"github.com/OdairPianta/unisinos_usp_api/models"
)

const chatTemplateFileMaxChars = 4000
const chatTemplateContextMaxFiles = 8

func BuildChatTemplateContextText(template models.ChatTemplate) string {
	processedFiles := make([]models.ChatTemplateFile, 0, len(template.ChatTemplateFiles))
	for _, file := range template.ChatTemplateFiles {
		if strings.TrimSpace(file.Content) == "" || file.Status == "failed" {
			continue
		}
		processedFiles = append(processedFiles, file)
	}
	if len(processedFiles) == 0 {
		return ""
	}
	if len(processedFiles) > chatTemplateContextMaxFiles {
		processedFiles = processedFiles[:chatTemplateContextMaxFiles]
	}
	var builder strings.Builder
	builder.WriteString("Fontes de contexto do template:\n")
	for _, file := range processedFiles {
		builder.WriteString("\n- Nome: ")
		builder.WriteString(file.Name)
		builder.WriteString("\n  Tipo: ")
		builder.WriteString(file.SourceType)
		if strings.TrimSpace(file.SourceURL) != "" {
			builder.WriteString("\n  URL: ")
			builder.WriteString(file.SourceURL)
		}
		if strings.TrimSpace(file.MimeType) != "" {
			builder.WriteString("\n  Mime-Type: ")
			builder.WriteString(file.MimeType)
		}
		builder.WriteString("\n  Conteúdo extraído:\n")
		builder.WriteString(truncateTextForContext(file.Content, chatTemplateFileMaxChars))
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func truncateTextForContext(value string, maxChars int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maxChars {
		return value
	}
	return fmt.Sprintf("%s... [truncado]", strings.TrimSpace(value[:maxChars]))
}
