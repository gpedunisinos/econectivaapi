package controllers

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/http/requests"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/repository"
	"github.com/OdairPianta/unisinos_usp_api/storagebackend"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const maxAttachmentBytes int64 = 25 * 1024 * 1024

var newAttachmentStore = storagebackend.New
var attachmentExtension = regexp.MustCompile(`^[a-z0-9]{1,10}$`)
var allowedAttachmentExtensions = strings.Fields("pdf txt csv json md rtf doc docx odt xls xlsx ods ppt pptx odp epub jpg jpeg png gif webp tif tiff mp3 wav wma aac flac ogg m4a mp4 mov webm zip")

// @Summary List attachments
// @Description Lists attachments, optionally scoped to one living clue. S3 download URLs are issued separately.
// @Tags files
// @Produce json
// @Param living_clue_id query int false "Living clue ID"
// @Success 200 {array} models.File
// @Failure 400 {object} models.APIMessage
// @Router /files [get]
// @Security Bearer
func FindFiles(c *gin.Context) {
	files := []models.File{}
	query := config.DB.Order("created_at DESC")
	if raw := c.Query("living_clue_id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || id == 0 {
			c.JSON(400, gin.H{"message": "Pista inválida."})
			return
		}
		query = query.Where("living_clue_id = ?", id)
	}
	if err := query.Find(&files).Error; err != nil {
		c.JSON(500, gin.H{"message": "Não foi possível carregar os arquivos."})
		return
	}
	c.JSON(200, files)
}

// @Summary Upload an attachment or save an external link
// @Description Multipart field file (up to 25 MB), or JSON base_64 for legacy clients, or public_url for an external link. New binary uploads use private S3 when STORAGE_DRIVER=s3.
// @Tags files
// @Accept multipart/form-data,json
// @Produce json
// @Param file formData file false "Attachment, max 25 MB"
// @Param name formData string false "Original filename"
// @Param extension formData string false "File extension"
// @Param living_clue_id formData int false "Living clue ID"
// @Param public_url formData string false "External HTTPS URL"
// @Success 200 {object} models.File
// @Failure 400 {object} models.APIMessage
// @Failure 413 {object} models.APIMessage
// @Failure 503 {object} models.APIMessage
// @Router /files [post]
// @Security Bearer
func CreateFile(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAttachmentBytes*4/3+1024*1024)
	input, data, err := parseAttachment(c)
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		attachmentInputError(c, err)
		return
	}
	if input.LivingClueID != nil {
		var clue models.LivingClue
		if *input.LivingClueID == 0 || config.DB.First(&clue, *input.LivingClueID).Error != nil {
			c.JSON(400, gin.H{"message": "A pista informada não está disponível."})
			return
		}
	}
	file := models.File{Extension: strings.ToLower(strings.TrimPrefix(input.Extension, ".")), Name: cleanAttachmentName(input.Name), LivingClueID: input.LivingClueID}
	if input.PublicURL != "" {
		if len(input.PublicURL) > 255 || !validExternalAttachmentURL(input.PublicURL) {
			c.JSON(400, gin.H{"message": "Informe um endereço HTTP ou HTTPS válido."})
			return
		}
		file.PublicURL = input.PublicURL
		file.Path = "" // Caller paths must never be used for deleting an external link.
	} else {
		if len(data) == 0 {
			c.JSON(400, gin.H{"message": "Selecione um arquivo que não esteja vazio."})
			return
		}
		if int64(len(data)) > maxAttachmentBytes {
			c.JSON(413, gin.H{"message": "O arquivo deve ter no máximo 25 MB."})
			return
		}
		if !supportedAttachment(file.Extension) {
			c.JSON(400, gin.H{"message": "Formato não permitido. Utilize documentos, imagens, áudio, vídeo ou ZIP."})
			return
		}
		file.Size = int64(len(data))
		file.MimeType = http.DetectContentType(data)
		if file.Name == "" {
			file.Name = "arquivo." + file.Extension
		}
		key := "files/" + time.Now().UTC().Format("2006/01/") + uuid.NewString() + "." + file.Extension
		ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
		defer cancel()
		if os.Getenv("STORAGE_DRIVER") == "s3" {
			store, err := newAttachmentStore(ctx)
			if err == nil {
				err = store.Put(ctx, key, data, file.MimeType, file.Name)
			}
			if err != nil {
				log.Error("Attachment storage upload failed")
				c.JSON(503, gin.H{"message": "Não foi possível armazenar o arquivo. Tente novamente."})
				return
			}
			file.StorageProvider = "s3"
			file.StorageKey = key
			file.StorageBucket = store.Bucket()
			if err = repository.CreateFile(&file); err != nil {
				cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
				defer stop()
				if store.Delete(cleanup, key) != nil {
					log.Error("Attachment orphan cleanup failed")
				}
				c.JSON(500, gin.H{"message": "Não foi possível vincular o arquivo à pista."})
				return
			}
			c.JSON(200, file)
			return
		}
		// Local storage remains available to isolated development and legacy installs.
		file.Path = key
		file.PublicURL, err = models.SaveGryphonAPI(base64.StdEncoding.EncodeToString(data), file.Extension, key)
		if err != nil {
			c.JSON(503, gin.H{"message": "Não foi possível armazenar o arquivo."})
			return
		}
	}
	if err := repository.CreateFile(&file); err != nil {
		c.JSON(500, gin.H{"message": "Não foi possível salvar o arquivo."})
		return
	}
	c.JSON(200, file)
}

func parseAttachment(c *gin.Context) (requests.CreateFileInput, []byte, error) {
	var input requests.CreateFileInput
	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		if err := c.Request.ParseMultipartForm(1024 * 1024); err != nil {
			return input, nil, err
		}
		input.Name = c.PostForm("name")
		input.Extension = c.PostForm("extension")
		input.PublicURL = strings.TrimSpace(c.PostForm("public_url"))
		if raw := c.PostForm("living_clue_id"); raw != "" {
			id, err := strconv.ParseUint(raw, 10, 64)
			if err != nil || id == 0 {
				return input, nil, errors.New("invalid clue")
			}
			value := uint(id)
			input.LivingClueID = &value
		}
		uploads := c.Request.MultipartForm.File["file"]
		if input.PublicURL != "" {
			if len(uploads) != 0 {
				return input, nil, errors.New("ambiguous upload")
			}
			return input, nil, nil
		}
		if len(uploads) != 1 {
			return input, nil, errors.New("one file required")
		}
		upload := uploads[0]
		if upload.Size > maxAttachmentBytes {
			return input, nil, &http.MaxBytesError{Limit: maxAttachmentBytes}
		}
		if input.Name == "" {
			input.Name = upload.Filename
		}
		// The actual filename determines the format; a supplied extension cannot disguise it.
		input.Extension = strings.TrimPrefix(strings.ToLower(filepath.Ext(upload.Filename)), ".")
		stream, err := upload.Open()
		if err != nil {
			return input, nil, err
		}
		defer stream.Close()
		data, err := io.ReadAll(io.LimitReader(stream, maxAttachmentBytes+1))
		return input, data, err
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		return input, nil, err
	}
	input.PublicURL = strings.TrimSpace(input.PublicURL)
	if input.PublicURL != "" {
		if input.Base64 != "" {
			return input, nil, errors.New("ambiguous upload")
		}
		return input, nil, nil
	}
	data, err := base64.StdEncoding.DecodeString(input.Base64)
	return input, data, err
}
func attachmentInputError(c *gin.Context, err error) {
	var size *http.MaxBytesError
	if errors.As(err, &size) {
		c.JSON(413, gin.H{"message": "O arquivo deve ter no máximo 25 MB."})
		return
	}
	c.JSON(400, gin.H{"message": "Envie um arquivo válido ou um link público."})
}
func supportedAttachment(ext string) bool {
	if !attachmentExtension.MatchString(ext) {
		return false
	}
	for _, allowed := range allowedAttachmentExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}
func cleanAttachmentName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	if name == "." || name == "/" {
		return ""
	}
	for len(name) > 240 {
		_, n := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-n]
	}
	return name
}
func validExternalAttachmentURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil
}

// @Summary Get temporary attachment download access
// @Description Requires a platform session. S3 links expire after five minutes and must not be persisted. Existing local files and external links remain supported.
// @Tags files
// @Produce json
// @Param id path int true "File ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} models.APIMessage
// @Failure 503 {object} models.APIMessage
// @Router /files/{id}/access [get]
// @Security Bearer
func FileAccess(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	file, err := repository.FindFileByID(parseFileID(c.Param("id")))
	if err != nil {
		c.JSON(404, gin.H{"message": "Arquivo não encontrado."})
		return
	}
	target := file.PublicURL
	if file.StorageProvider == "s3" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		store, e := newAttachmentStore(ctx)
		if e != nil || store.Bucket() != file.StorageBucket {
			c.JSON(503, gin.H{"message": "Armazenamento indisponível."})
			return
		}
		target, e = store.URL(ctx, file.StorageKey, file.Name)
		if e != nil {
			c.JSON(503, gin.H{"message": "Não foi possível acessar o arquivo."})
			return
		}
	}
	if !validExternalAttachmentURL(target) {
		c.JSON(404, gin.H{"message": "Endereço do arquivo indisponível."})
		return
	}
	c.JSON(200, gin.H{"url": target, "expires_in": 300})
}
func parseFileID(raw string) uint { v, _ := strconv.ParseUint(raw, 10, 64); return uint(v) }

func isLocalAttachment(file *models.File) bool {
	base := strings.TrimRight(os.Getenv("APP_URL"), "/")
	if base == "" {
		base = "http://localhost:8080"
	}
	return file.PublicURL == base+"/storage/"+filepath.ToSlash(filepath.Clean(file.Path))
}

// @Summary Remove an attachment
// @Description Removes a linked attachment. S3 versioning retains recoverable previous versions. External links never delete local files.
// @Tags files
// @Produce json
// @Param id path int true "File ID"
// @Success 200 {object} models.File
// @Failure 404 {object} models.APIMessage
// @Failure 503 {object} models.APIMessage
// @Router /files/{id} [delete]
// @Security Bearer
func DeleteFile(c *gin.Context) {
	file, err := repository.FindFileByID(parseFileID(c.Param("id")))
	if err != nil {
		c.JSON(404, gin.H{"message": "Arquivo não encontrado."})
		return
	}
	if file.StorageProvider == "s3" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		store, e := newAttachmentStore(ctx)
		if e != nil || store.Bucket() != file.StorageBucket {
			c.JSON(503, gin.H{"message": "Armazenamento indisponível."})
			return
		}
		err = config.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Delete(file).Error; err != nil {
				return err
			}
			return store.Delete(ctx, file.StorageKey)
		})
	} else {
		err = repository.DeleteFile(file)
		if err == nil && file.Path != "" && isLocalAttachment(file) {
			cleaned := filepath.Clean(file.Path)
			if !filepath.IsAbs(cleaned) && cleaned != ".." && !strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
				_ = os.Remove(filepath.Join("storage", cleaned))
			}
		}
	}
	if err != nil {
		c.JSON(503, gin.H{"message": "Não foi possível remover o arquivo. Tente novamente."})
		return
	}
	c.JSON(200, file)
}
