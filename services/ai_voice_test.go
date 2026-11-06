package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testVoiceAudio() []byte {
	p := make([]byte, 32000)
	for i := 0; i < len(p); i += 2 {
		binary.LittleEndian.PutUint16(p[i:], 1000)
	}
	return VoiceWAV(p, 16000)
}
func TestVoiceValidatesActualPCMNotUploadMetadata(t *testing.T) {
	good := testVoiceAudio()
	require.NoError(t, ValidateVoiceWAV(good))
	for _, b := range [][]byte{nil, []byte("RIFF"), VoiceWAV(make([]byte, 32000), 16000), VoiceWAV(make([]byte, 32000), 24000), good[:100], append(append([]byte{}, good...), 0)} {
		require.Error(t, ValidateVoiceWAV(b))
	}
	corrupt := append([]byte{}, good...)
	corrupt[22] = 2
	require.Error(t, ValidateVoiceWAV(corrupt))
}
func TestVoiceTranscriptionWireAndSafeFailure(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-private-key")
	t.Setenv("OPENAI_TRANSCRIBE_MODEL", "gpt-transcribe")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/audio/transcriptions", r.URL.Path)
		require.Equal(t, "Bearer test-private-key", r.Header.Get("Authorization"))
		require.NoError(t, r.ParseMultipartForm(5<<20))
		require.Equal(t, "gpt-transcribe", r.FormValue("model"))
		require.Equal(t, "pt", r.FormValue("languages[]"))
		require.Contains(t, r.MultipartForm.Value["keywords[]"], "IAraeté")
		require.Contains(t, r.MultipartForm.Value["keywords[]"], "Educação OnLIFE")
		f, _, err := r.FormFile("file")
		require.NoError(t, err)
		defer f.Close()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"text":" Como explorar a Pistas em fluxo? "}`))
	}))
	defer server.Close()
	s := OpenAIVoiceService{BaseURL: server.URL}
	text, err := s.Transcribe(context.Background(), testVoiceAudio())
	require.NoError(t, err)
	require.Equal(t, "Como explorar a Pistas em fluxo?", text)
	t.Setenv("OPENAI_API_KEY", "")
	_, err = s.Transcribe(context.Background(), testVoiceAudio())
	require.ErrorIs(t, err, ErrVoiceUnavailable)
}
func TestVoiceProviderErrorNeverExposesCredentialsOrPayload(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-private-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"secret and full recording"}`))
	}))
	defer server.Close()
	_, err := (&OpenAIVoiceService{BaseURL: server.URL}).Transcribe(context.Background(), testVoiceAudio())
	require.ErrorIs(t, err, ErrVoiceProvider)
	require.NotContains(t, err.Error(), "secret")
}
func TestVoiceRealtimeNarratesExistingReplyAndWrapsPCM(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-private-key")
	t.Setenv("OPENAI_VOICE_MODEL", "gpt-realtime-2.1-mini")
	t.Setenv("OPENAI_VOICE", "marin")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "gpt-realtime-2.1-mini", r.URL.Query().Get("model"))
		require.Equal(t, "Bearer test-private-key", r.Header.Get("Authorization"))
		c, err := websocket.Accept(w, r, nil)
		require.NoError(t, err)
		defer c.CloseNow()
		ctx := r.Context()
		var session map[string]interface{}
		require.NoError(t, wsjson.Read(ctx, c, &session))
		require.Equal(t, "session.update", session["type"])
		require.NoError(t, wsjson.Write(ctx, c, map[string]string{"type": "session.updated"}))
		var req map[string]interface{}
		require.NoError(t, wsjson.Read(ctx, c, &req))
		raw, _ := json.Marshal(req)
		require.True(t, strings.Contains(string(raw), "Resposta já salva"))
		require.Contains(t, string(raw), `"conversation":"none"`)
		for _, e := range []map[string]interface{}{{"type": "response.output_audio.delta", "delta": base64.StdEncoding.EncodeToString([]byte{10, 0, 11, 0})}, {"type": "response.done", "response": map[string]string{"status": "completed"}}} {
			require.NoError(t, wsjson.Write(ctx, c, e))
		}
	}))
	defer server.Close()
	audio, err := (&OpenAIVoiceService{BaseURL: server.URL}).Speak(context.Background(), "Resposta já salva")
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(audio, []byte("RIFF")))
	require.Equal(t, uint32(24000), binary.LittleEndian.Uint32(audio[24:]))
	require.Equal(t, []byte{10, 0, 11, 0}, audio[44:])
}
