package models

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type File struct {
	StorageProvider string `gorm:"size:20" json:"storage_provider,omitempty"`
	StorageKey      string `gorm:"size:255" json:"-"`
	StorageBucket   string `gorm:"size:255" json:"-"`
	Size            int64  `json:"size"`
	MimeType        string `gorm:"size:128" json:"mime_type,omitempty"`
	DefaultModel
	Extension    string      `gorm:"not null" validate:"required" json:"extension"`
	Path         string      `gorm:"size:255;null" json:"path"`
	PublicURL    string      `gorm:"size:255;null" json:"public_url"`
	Name         string      `gorm:"size:255;null" json:"name"`
	PostID       *uint       `gorm:"null" json:"post_id"`
	LivingClueID *uint       `gorm:"index" json:"living_clue_id"`
	LivingClue   *LivingClue `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"living_clue"`
}

func SaveGryphonAPI(base64Value, extension, filePath string) (string, error) {
	_, publicURL, err := saveBase64ToLocalStorage(base64Value, extension, filePath)
	if err != nil {
		return "", err
	}
	return publicURL, nil
}

func SaveImageGryphonAPI(base64Value, extension, filePath string, width, height int) (string, error) {
	_, publicURL, err := saveBase64ToLocalStorage(base64Value, extension, filePath)
	if err != nil {
		return "", err
	}
	return publicURL, nil
}

func saveBase64ToLocalStorage(base64Value, extension, filePath string) (string, string, error) {
	data, err := base64.StdEncoding.DecodeString(base64Value)
	if err != nil {
		return "", "", err
	}
	return saveBytesToLocalStorage(data, extension, filePath)
}

func saveBytesToLocalStorage(data []byte, extension, filePath string) (string, string, error) {
	extension = strings.TrimPrefix(strings.TrimSpace(extension), ".")
	if extension == "" {
		extension = "bin"
	}
	relativePath := sanitizeStorageRelativePath(filePath, extension)
	absolutePath := filepath.Join("storage", relativePath)
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(absolutePath, data, 0o644); err != nil {
		return "", "", err
	}
	appURL := strings.TrimRight(os.Getenv("APP_URL"), "/")
	if appURL == "" {
		appURL = "http://localhost:8080"
	}
	publicURL := fmt.Sprintf("%s/storage/%s", appURL, filepath.ToSlash(relativePath))
	return relativePath, publicURL, nil
}

func sanitizeStorageRelativePath(filePath string, extension string) string {
	cleaned := filepath.Clean(strings.TrimSpace(filePath))
	cleaned = strings.TrimPrefix(cleaned, "/")
	cleaned = strings.TrimPrefix(cleaned, "./")
	if cleaned == "." || cleaned == "" {
		cleaned = fmt.Sprintf("uploads/%d.%s", time.Now().UnixNano(), extension)
	}
	if strings.HasSuffix(cleaned, "/") || filepath.Ext(cleaned) == "" {
		cleaned = filepath.Join(cleaned, fmt.Sprintf("%d.%s", time.Now().UnixNano(), extension))
	}
	return cleaned
}
