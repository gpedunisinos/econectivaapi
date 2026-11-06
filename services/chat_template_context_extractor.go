package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/models"
)

const (
	chatTemplateSourceTypeText = "text"
	chatTemplateSourceTypeURL  = "url"
	chatTemplateSourceTypeFile = "file"
)

type ChatTemplateExtractionResult struct {
	OriginalContent  string
	ExtractedContent string
	MimeType         string
	Status           string
	ErrorMessage     string
}

type ChatTemplateExtractor struct {
	client *http.Client
}

func NewChatTemplateExtractor() *ChatTemplateExtractor {
	return &ChatTemplateExtractor{client: &http.Client{Timeout: 20 * time.Second}}
}

func (s *ChatTemplateExtractor) ProcessInput(input models.ChatTemplateFile) models.ChatTemplateFile {
	result := ChatTemplateExtractionResult{Status: "processed", MimeType: input.MimeType}
	sourceType := strings.TrimSpace(input.SourceType)
	if sourceType == "" {
		sourceType = inferChatTemplateSourceType(input)
	}
	input.SourceType = sourceType

	switch sourceType {
	case chatTemplateSourceTypeText:
		result.OriginalContent = input.OriginalContent
		if strings.TrimSpace(result.OriginalContent) == "" {
			result.OriginalContent = input.Content
		}
		result.ExtractedContent = normalizeExtractedText(result.OriginalContent)
		if strings.TrimSpace(result.MimeType) == "" {
			result.MimeType = "text/plain"
		}
	case chatTemplateSourceTypeURL:
		result = s.extractFromRemoteURL(input.SourceURL, "", "", 8*1024*1024)
	case chatTemplateSourceTypeFile:
		result = s.extractFromRemoteURL(input.SourceURL, input.MimeType, input.Name, 12*1024*1024)
	default:
		result.Status = "failed"
		result.ErrorMessage = "Tipo de fonte não suportado"
	}

	input.OriginalContent = result.OriginalContent
	input.Content = result.ExtractedContent
	input.ExtractedContent = result.ExtractedContent
	input.MimeType = firstNonEmpty(result.MimeType, input.MimeType)
	input.Status = firstNonEmpty(result.Status, "processed")
	input.ErrorMessage = result.ErrorMessage
	return input
}

func (s *ChatTemplateExtractor) ProcessUploadedFile(sourceName string, mimeType string, body []byte) ChatTemplateExtractionResult {
	resolvedMimeType := firstNonEmpty(mimeType, http.DetectContentType(body), mimeTypeFromExtension(filepath.Ext(sourceName)))
	return s.extractFromBytes(body, resolvedMimeType, sourceName)
}

func inferChatTemplateSourceType(input models.ChatTemplateFile) string {
	if strings.TrimSpace(input.SourceURL) != "" {
		return chatTemplateSourceTypeURL
	}
	return chatTemplateSourceTypeText
}

func (s *ChatTemplateExtractor) extractFromRemoteURL(raw string, preferredMimeType string, sourceName string, bodyLimit int64) ChatTemplateExtractionResult {
	parsed, err := validateSafeURL(raw)
	if err != nil {
		return ChatTemplateExtractionResult{Status: "failed", ErrorMessage: err.Error()}
	}
	resp, err := s.client.Get(parsed.String())
	if err != nil {
		return ChatTemplateExtractionResult{Status: "failed", ErrorMessage: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return ChatTemplateExtractionResult{Status: "failed", ErrorMessage: fmt.Sprintf("falha ao buscar URL: status %d", resp.StatusCode)}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, bodyLimit))
	if err != nil {
		return ChatTemplateExtractionResult{Status: "failed", ErrorMessage: err.Error()}
	}
	mimeType := firstNonEmpty(preferredMimeType, strings.TrimSpace(resp.Header.Get("Content-Type")), http.DetectContentType(body))
	resolvedSourceName := firstNonEmpty(sourceName, filepath.Base(parsed.Path))
	return s.extractFromBytes(body, mimeType, resolvedSourceName)
}

func (s *ChatTemplateExtractor) extractFromBytes(body []byte, mimeType string, sourceName string) ChatTemplateExtractionResult {
	if IsImageMimeType(mimeType) {
		return s.extractFromImageBytes(body, mimeType, sourceName)
	}
	text, original, err := extractTextFromBytes(body, mimeType, sourceName)
	if err != nil {
		return ChatTemplateExtractionResult{Status: "failed", ErrorMessage: err.Error(), MimeType: mimeType}
	}
	return ChatTemplateExtractionResult{Status: "processed", MimeType: mimeType, OriginalContent: original, ExtractedContent: text}
}

func (s *ChatTemplateExtractor) extractFromImageBytes(body []byte, mimeType string, sourceName string) ChatTemplateExtractionResult {
	optimizedBytes, optimizedMimeType, err := OptimizeImageToJPEG(body)
	if err != nil {
		return ChatTemplateExtractionResult{Status: "failed", ErrorMessage: err.Error(), MimeType: mimeType}
	}
	description, err := GetChatTemplateImageDescriber().DescribeImage(
		context.Background(),
		optimizedBytes,
		optimizedMimeType,
		firstNonEmpty(sourceName, "chat-template-image.jpg"),
	)
	if err != nil {
		return ChatTemplateExtractionResult{Status: "failed", ErrorMessage: err.Error(), MimeType: optimizedMimeType}
	}
	return ChatTemplateExtractionResult{
		Status:           "processed",
		MimeType:         optimizedMimeType,
		ExtractedContent: normalizeExtractedText(description),
	}
}

func extractTextFromBytes(body []byte, mimeType string, sourceName string) (string, string, error) {
	mimeType = strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	original := string(body)
	ext := strings.ToLower(filepath.Ext(sourceName))

	switch {
	case strings.Contains(mimeType, "json") || ext == ".json":
		var decoded interface{}
		if err := json.Unmarshal(body, &decoded); err == nil {
			pretty, _ := json.MarshalIndent(decoded, "", "  ")
			original = string(pretty)
			return normalizeExtractedText(original), original, nil
		}
		return normalizeExtractedText(original), original, nil
	case strings.Contains(mimeType, "markdown") || ext == ".md":
		return normalizeExtractedText(original), original, nil
	case strings.Contains(mimeType, "csv") || ext == ".csv":
		return normalizeExtractedText(original), original, nil
	case strings.Contains(mimeType, "plain") || ext == ".txt":
		return normalizeExtractedText(original), original, nil
	case strings.Contains(mimeType, "html") || ext == ".html" || ext == ".htm":
		text := stripHTML(string(body))
		return normalizeExtractedText(text), string(body), nil
	case strings.Contains(mimeType, "pdf") || ext == ".pdf":
		text, err := extractPDFText(body)
		return normalizeExtractedText(text), "", err
	case strings.Contains(mimeType, "wordprocessingml") || ext == ".docx":
		text, err := extractDOCXText(body)
		return normalizeExtractedText(text), "", err
	case strings.Contains(mimeType, "msword") || ext == ".doc":
		return "", "", errors.New("Extração de DOC legada ainda não suportada; converta para DOCX ou PDF")
	default:
		if strings.HasPrefix(mimeType, "text/") {
			return normalizeExtractedText(original), original, nil
		}
		return "", "", fmt.Errorf("Tipo de arquivo não suportado para extração: %s", mimeType)
	}
}

func extractPDFText(body []byte) (string, error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return "", errors.New("pdftotext não disponível no servidor")
	}
	inputFile, err := os.CreateTemp("", "chat-template-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(inputFile.Name())
	if _, err := inputFile.Write(body); err != nil {
		inputFile.Close()
		return "", err
	}
	inputFile.Close()
	outputFile := inputFile.Name() + ".txt"
	defer os.Remove(outputFile)
	cmd := exec.Command("pdftotext", inputFile.Name(), outputFile)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	text, err := os.ReadFile(outputFile)
	if err != nil {
		return "", err
	}
	return string(text), nil
}

func extractDOCXText(body []byte) (string, error) {
	readerAt := bytes.NewReader(body)
	zipReader, err := zip.NewReader(readerAt, int64(len(body)))
	if err != nil {
		return "", err
	}
	for _, file := range zipReader.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return "", err
		}
		xmlText := string(data)
		xmlText = strings.ReplaceAll(xmlText, "</w:p>", "\n")
		xmlText = strings.ReplaceAll(xmlText, "</w:tr>", "\n")
		return stripXMLTags(xmlText), nil
	}
	return "", errors.New("Arquivo DOCX sem word/document.xml")
}

func normalizeExtractedText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	cleaned := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			blankCount++
			if blankCount > 1 {
				continue
			}
		} else {
			blankCount = 0
		}
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func stripHTML(value string) string {
	value = regexp.MustCompile(`(?is)<script.*?>.*?</script>`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?is)<style.*?>.*?</style>`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?i)</p>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?s)<[^>]+>`).ReplaceAllString(value, " ")
	return htmlEntityCleanup(value)
}

func stripXMLTags(value string) string {
	value = regexp.MustCompile(`(?s)<[^>]+>`).ReplaceAllString(value, " ")
	return htmlEntityCleanup(value)
}

func htmlEntityCleanup(value string) string {
	replacer := strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'")
	return normalizeExtractedText(replacer.Replace(value))
}

func validateSafeURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("Apenas URLs http/https são permitidas")
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		return nil, errors.New("URL inválida")
	}
	if os.Getenv("ALLOW_PRIVATE_CONTEXT_URLS") != "1" {
		ips, err := net.LookupIP(hostname)
		if err == nil {
			for _, ip := range ips {
				if isPrivateIP(ip) {
					return nil, errors.New("URL em rede privada não é permitida")
				}
			}
		}
	}
	return parsed, nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
		return true
	}
	return false
}

func mimeTypeFromExtension(ext string) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "pdf":
		return "application/pdf"
	case "md":
		return "text/markdown"
	case "txt":
		return "text/plain"
	case "json":
		return "application/json"
	case "csv":
		return "text/csv"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "doc":
		return "application/msword"
	case "html", "htm":
		return "text/html"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
