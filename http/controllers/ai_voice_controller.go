package controllers

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
)

// Bound provider use; no audio or text is retained in the limiter.
var voiceSlots = make(chan struct{}, 4)
var voiceMu sync.Mutex
var voiceUsers = map[uint]struct {
	Since time.Time
	Count int
	Busy  bool
}{}

func reserveVoice(c *gin.Context) func() {
	id := c.MustGet("user").(models.User).ID
	now := time.Now()
	voiceMu.Lock()
	for k, v := range voiceUsers {
		if !v.Busy && now.Sub(v.Since) > time.Minute {
			delete(voiceUsers, k)
		}
	}
	v := voiceUsers[id]
	if v.Busy || v.Count >= 20 {
		voiceMu.Unlock()
		c.Header("Retry-After", "10")
		c.JSON(429, gin.H{"message": "Aguarde um momento antes de solicitar outro áudio."})
		return nil
	}
	select {
	case voiceSlots <- struct{}{}:
	default:
		voiceMu.Unlock()
		c.JSON(429, gin.H{"message": "O canal de voz está ocupado. Tente em instantes."})
		return nil
	}
	if v.Since.IsZero() {
		v.Since = now
	}
	v.Count++
	v.Busy = true
	voiceUsers[id] = v
	voiceMu.Unlock()
	return func() {
		<-voiceSlots
		voiceMu.Lock()
		v := voiceUsers[id]
		v.Busy = false
		voiceUsers[id] = v
		voiceMu.Unlock()
	}
}
func voiceError(c *gin.Context, err error) {
	status := http.StatusBadGateway
	if errors.Is(err, services.ErrVoiceUnavailable) {
		status = 503
	}
	if errors.Is(err, services.ErrVoiceEmpty) {
		status = 422
	}
	c.JSON(status, gin.H{"message": err.Error()})
}

// TranscribeAIVoice godoc
// @Summary Transcrever gravação da conversa com a IAraeté
// @Description Recebe WAV PCM16 mono 16kHz entre 0,5 e 120 segundos. Processa em memória e descarta o áudio. Não salva mensagens; a transcrição segue pelo endpoint normal de conversa.
// @Tags ai_messages
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Gravação WAV"
// @Success 200 {object} map[string]string
// @Failure 422 {object} models.APIMessage
// @Failure 503 {object} models.APIMessage
// @Router /ai/voice/transcriptions [post]
// @Security Bearer
func TranscribeAIVoice(c *gin.Context) {
	release := reserveVoice(c)
	if release == nil {
		return
	}
	defer release()
	c.Header("Cache-Control", "no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.VoiceMaxUploadBytes+64*1024)
	if err := c.Request.ParseMultipartForm(services.VoiceMaxUploadBytes + 64*1024); err != nil {
		c.JSON(413, gin.H{"message": "Envie uma gravação de até 2 minutos."})
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	files := c.Request.MultipartForm.File["file"]
	if len(files) != 1 || len(c.Request.MultipartForm.File) != 1 {
		c.JSON(400, gin.H{"message": "Envie uma única gravação."})
		return
	}
	f, err := files[0].Open()
	if err != nil {
		c.JSON(400, gin.H{"message": "Gravação inválida."})
		return
	}
	defer f.Close()
	audio, err := io.ReadAll(io.LimitReader(f, services.VoiceMaxUploadBytes+1))
	if err != nil {
		c.JSON(400, gin.H{"message": "Gravação inválida."})
		return
	}
	if err = services.ValidateVoiceWAV(audio); err != nil {
		voiceError(c, err)
		return
	}
	text, err := services.GetVoiceService().Transcribe(c.Request.Context(), audio)
	if err != nil {
		voiceError(c, err)
		return
	}
	c.JSON(200, gin.H{"text": text})
}

type AISpeechInput struct {
	MessageID uint `json:"message_id" binding:"required"`
}

// SpeakAIMessage godoc
// @Summary Ouvir uma resposta da IAraeté
// @Description Sintetiza somente uma mensagem assistant do próprio usuário. Voz gerada por IA; áudio não é armazenado. A falha de voz não altera o histórico de texto.
// @Tags ai_messages
// @Accept json
// @Produce audio/wav
// @Param input body AISpeechInput true "Mensagem da IAraeté"
// @Success 200 {file} binary
// @Failure 404 {object} models.APIMessage
// @Failure 503 {object} models.APIMessage
// @Router /ai/voice/speech [post]
// @Security Bearer
func SpeakAIMessage(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	var input AISpeechInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(400, gin.H{"message": "Selecione uma resposta da IAraeté."})
		return
	}
	user := c.MustGet("user").(models.User)
	var message models.AIMessage
	if config.DB.Where("id = ? AND user_id = ? AND role = ?", input.MessageID, user.ID, "assistant").First(&message).Error != nil {
		c.JSON(404, gin.H{"message": "Resposta não encontrada."})
		return
	}
	release := reserveVoice(c)
	if release == nil {
		return
	}
	defer release()
	audio, err := services.GetVoiceService().Speak(c.Request.Context(), message.Content)
	if err != nil {
		voiceError(c, err)
		return
	}
	c.Data(200, "audio/wav", audio)
}
