package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/OdairPianta/unisinos_usp_api/config"
	"github.com/OdairPianta/unisinos_usp_api/models"
	"github.com/OdairPianta/unisinos_usp_api/tests/factories"
	"github.com/stretchr/testify/require"
)

func TestFileS3UploadAccessAndRemoval(t *testing.T) {
	setupDatabase()
	var mu sync.Mutex
	objects := map[string][]byte{}
	s3server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case "PUT":
			require.Equal(t, "AES256", r.Header.Get("X-Amz-Server-Side-Encryption"))
			require.Contains(t, r.Header.Get("Content-Disposition"), "attachment")
			data, _ := io.ReadAll(r.Body)
			objects[r.URL.Path] = data
			w.Header().Set("ETag", `"synthetic-etag"`)
			w.WriteHeader(200)
		case "DELETE":
			delete(objects, r.URL.Path)
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	defer s3server.Close()
	t.Setenv("STORAGE_DRIVER", "s3")
	t.Setenv("S3_BUCKET", "onlife-test")
	t.Setenv("AWS_ACCESS_KEY_ID", "synthetic-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "synthetic-secret")
	t.Setenv("AWS_ENDPOINT_URL_S3", s3server.URL)
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	router := routesSetup()
	user, token := initUser()
	clue, err := factories.CreateLivingClue()
	require.NoError(t, err)
	other, err := factories.CreateLivingClue()
	require.NoError(t, err)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "evidência.txt")
	require.NoError(t, err)
	part.Write([]byte("Observação de teste"))
	writer.WriteField("name", "evidência.txt")
	writer.WriteField("living_clue_id", fmt.Sprint(clue.ID))
	writer.WriteField("path", "../../.env")
	writer.Close()
	request := httptest.NewRequest("POST", "/api/files", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, 200, response.Code, response.Body.String())
	var file models.File
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &file))
	require.Equal(t, "s3", file.StorageProvider)
	require.Empty(t, file.PublicURL)
	require.Empty(t, file.Path)
	require.Equal(t, int64(len("Observação de teste")), file.Size)
	require.NotContains(t, response.Body.String(), "storage_key")
	require.NotContains(t, response.Body.String(), "storage_bucket")
	mu.Lock()
	require.Len(t, objects, 1)
	mu.Unlock()
	var stored models.File
	require.NoError(t, config.DB.First(&stored, file.ID).Error)
	require.True(t, strings.HasPrefix(stored.StorageKey, "files/"))
	require.Equal(t, "onlife-test", stored.StorageBucket)

	get := func(path, auth string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if auth != "" {
			req.Header.Set("Authorization", "Bearer "+auth)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	require.Equal(t, 401, get(fmt.Sprintf("/api/files/%d/access", file.ID), "").Code)
	access := get(fmt.Sprintf("/api/files/%d/access", file.ID), token)
	require.Equal(t, 200, access.Code, access.Body.String())
	require.Equal(t, "no-store", access.Header().Get("Cache-Control"))
	var grant map[string]interface{}
	json.Unmarshal(access.Body.Bytes(), &grant)
	signed, err := url.Parse(grant["url"].(string))
	require.NoError(t, err)
	require.Equal(t, "300", signed.Query().Get("X-Amz-Expires"))
	require.NotEmpty(t, signed.Query().Get("X-Amz-Signature"))
	require.Contains(t, signed.Query().Get("response-content-disposition"), "attachment")
	scoped := get(fmt.Sprintf("/api/files?living_clue_id=%d", other.ID), token)
	require.Equal(t, "[]", strings.TrimSpace(scoped.Body.String()))
	listed := get(fmt.Sprintf("/api/files?living_clue_id=%d", clue.ID), token)
	require.Contains(t, listed.Body.String(), "evidência.txt")
	require.NotContains(t, listed.Body.String(), "X-Amz-Signature")
	explorer := "explorer"
	require.NoError(t, config.DB.Model(&user).Update("usage_mode", explorer).Error)
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/files/%d", file.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, req)
	require.Equal(t, 403, denied.Code)
	require.Equal(t, 200, get(fmt.Sprintf("/api/files/%d/access", file.ID), token).Code)
	require.NoError(t, config.DB.Model(&user).Update("usage_mode", "inventor").Error)
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/files/%d", file.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	removed := httptest.NewRecorder()
	router.ServeHTTP(removed, req)
	require.Equal(t, 200, removed.Code, removed.Body.String())
	mu.Lock()
	require.Empty(t, objects)
	mu.Unlock()
	require.Equal(t, 404, get(fmt.Sprintf("/api/files/%d/access", file.ID), token).Code)
}

func TestFileUploadValidationAndExternalLinks(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	for _, tt := range []struct {
		name   string
		data   []byte
		status int
	}{
		{"executavel.html", []byte("<html>test</html>"), 400},
		{"vazio.txt", nil, 400},
		{"grande.txt", make([]byte, 25*1024*1024+1), 413},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			w := multipart.NewWriter(&body)
			p, _ := w.CreateFormFile("file", tt.name)
			p.Write(tt.data)
			w.Close()
			req := httptest.NewRequest("POST", "/api/files", &body)
			req.Header.Set("Content-Type", w.FormDataContentType())
			req.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			require.Equal(t, tt.status, response.Code, response.Body.String())
		})
	}
	for _, raw := range []string{"javascript:alert(1)", "https://user:secret@example.com/a"} {
		input, _ := json.Marshal(map[string]interface{}{"extension": "link", "public_url": raw})
		req := httptest.NewRequest("POST", "/api/files", bytes.NewReader(input))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, 400, response.Code)
	}
}

func TestFileS3FailureLeavesNoAttachment(t *testing.T) {
	setupDatabase()
	router := routesSetup()
	_, token := initUser()
	t.Setenv("STORAGE_DRIVER", "s3")
	t.Setenv("S3_BUCKET", "")
	var before int64
	config.DB.Model(&models.File{}).Count(&before)
	req := httptest.NewRequest("POST", "/api/files", strings.NewReader(`{"base_64":"dGVzdGU=","extension":"txt","name":"teste.txt"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	require.Equal(t, 503, response.Code)
	var after int64
	config.DB.Model(&models.File{}).Count(&after)
	require.Equal(t, before, after)
}
