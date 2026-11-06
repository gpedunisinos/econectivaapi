package tests

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/stretchr/testify/require"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

type fakeVoice struct {
	Transcripts int
	Spoken      []string
}

func (f *fakeVoice) Transcribe(context.Context, []byte) (string, error) {
	f.Transcripts++
	return "Como explorar?", nil
}
func (f *fakeVoice) Speak(_ context.Context, text string) ([]byte, error) {
	f.Spoken = append(f.Spoken, text)
	return []byte("RIFFaudio"), nil
}
func TestVoiceAuthenticatedExplorerOwnMessagesOnly(t *testing.T) {
	setupDatabase()
	r := routesSetup()
	user, token := initUser()
	_, other := initUser()
	mode := "explorer"
	require.NoError(t, config.DB.Model(&user).Update("usage_mode", mode).Error)
	fake := &fakeVoice{}
	defer services.SetVoiceService(fake)()
	assistant := models.AIMessage{UserID: user.ID, Role: "assistant", Content: "Olá, explorador!"}
	require.NoError(t, config.DB.Create(&assistant).Error)
	ownUser := models.AIMessage{UserID: user.ID, Role: "user", Content: "não narrar"}
	require.NoError(t, config.DB.Create(&ownUser).Error)
	call := func(auth string, id uint) int {
		body, _ := json.Marshal(map[string]uint{"message_id": id})
		req := httptest.NewRequest("POST", "/api/ai/voice/speech", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if auth != "" {
			req.Header.Set("Authorization", "Bearer "+auth)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	require.Equal(t, 401, call("", assistant.ID))
	require.Equal(t, 404, call(other, assistant.ID))
	require.Equal(t, 404, call(token, ownUser.ID))
	require.Equal(t, 200, call(token, assistant.ID))
	require.Equal(t, []string{assistant.Content}, fake.Spoken)
	var body bytes.Buffer
	m := multipart.NewWriter(&body)
	f, _ := m.CreateFormFile("file", "arbitrario.wav")
	pcm := make([]byte, 32000)
	for i := 0; i < len(pcm); i += 2 {
		binary.LittleEndian.PutUint16(pcm[i:], 1000)
	}
	f.Write(services.VoiceWAV(pcm, 16000))
	m.Close()
	req := httptest.NewRequest("POST", "/api/ai/voice/transcriptions", &body)
	req.Header.Set("Content-Type", m.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "Como explorar?")
	require.Equal(t, 1, fake.Transcripts)
	var count int64
	config.DB.Model(&models.AIMessage{}).Where("user_id = ?", user.ID).Count(&count)
	require.Equal(t, int64(2), count, "Transcription must not create chat messages")
}
func TestVoiceRejectsInvalidAudioBeforeProvider(t *testing.T) {
	setupDatabase()
	r := routesSetup()
	_, token := initUser()
	fake := &fakeVoice{}
	defer services.SetVoiceService(fake)()
	var body bytes.Buffer
	m := multipart.NewWriter(&body)
	f, _ := m.CreateFormFile("file", "valid.wav")
	f.Write([]byte("not audio"))
	m.Close()
	req := httptest.NewRequest("POST", "/api/ai/voice/transcriptions", &body)
	req.Header.Set("Content-Type", m.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 422, w.Code)
	require.Zero(t, fake.Transcripts)
}
