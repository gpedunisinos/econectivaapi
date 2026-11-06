package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testLogAccessHash = "local-log-test-credential"

func TestLogEndpointDisabledWithoutConfiguredToken(t *testing.T) {
	setLogEnvironment(t, t.TempDir())
	t.Setenv("API_LOG_ACCESS_TOKEN", "")
	recorder := performLogRequest(t, "/api/log/"+testLogAccessHash)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestLogEndpointRejectsInvalidHash(t *testing.T) {
	setLogEnvironment(t, t.TempDir())
	recorder := performLogRequest(t, "/api/log/invalid")

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, "Not Found", recorder.Body.String())
}

func TestLogEndpointUsesCurrentDateByDefault(t *testing.T) {
	dir := t.TempDir()
	setLogEnvironment(t, dir)
	writeLogFixture(t, services.LogFilePath(time.Now()), 550)

	recorder := performLogRequest(t, "/api/log/"+testLogAccessHash)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "line 350")
	assert.Contains(t, recorder.Body.String(), "line 351")
	assert.Contains(t, recorder.Body.String(), "line 550")
	assert.Equal(t, 200, countLogLines(recorder.Body.String()))
}

func TestLogEndpointSupportsDateLimitAndOffset(t *testing.T) {
	dir := t.TempDir()
	setLogEnvironment(t, dir)
	date := time.Date(2026, time.August, 4, 0, 0, 0, 0, time.Local)
	writeLogFixture(t, services.LogFilePath(date), 550)

	recorder := performLogRequest(t, "/api/log/"+testLogAccessHash+"?date=2026-08-04&limit=200&offset=200")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/plain; charset=UTF-8", recorder.Header().Get("Content-Type"))
	assert.NotContains(t, recorder.Body.String(), "line 150")
	assert.Contains(t, recorder.Body.String(), "line 151")
	assert.Contains(t, recorder.Body.String(), "line 350")
	assert.NotContains(t, recorder.Body.String(), "line 351")
	assert.Equal(t, 200, countLogLines(recorder.Body.String()))
}

func TestLogEndpointRejectsInvalidParameters(t *testing.T) {
	setLogEnvironment(t, t.TempDir())

	for _, query := range []string{"date=04-08-2026", "limit=0", "limit=1001", "limit=abc", "offset=-1"} {
		recorder := performLogRequest(t, "/api/log/"+testLogAccessHash+"?"+query)
		assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code, query)
	}
}

func TestLogEndpointReturnsNotFoundForMissingFile(t *testing.T) {
	setLogEnvironment(t, t.TempDir())
	recorder := performLogRequest(t, "/api/log/"+testLogAccessHash+"?date=2026-08-03")
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestDailyLogWriterWritesContinuouslyAndRemovesExpiredFiles(t *testing.T) {
	dir := t.TempDir()
	prefix := "application"
	now := time.Now()
	expiredPath := filepath.Join(dir, prefix+"-"+now.AddDate(0, 0, -15).Format("2006-01-02")+".log")
	retainedPath := filepath.Join(dir, prefix+"-"+now.AddDate(0, 0, -14).Format("2006-01-02")+".log")
	require.NoError(t, os.WriteFile(expiredPath, []byte("expired\n"), 0o664))
	require.NoError(t, os.WriteFile(retainedPath, []byte("retained\n"), 0o664))

	writer, err := services.NewDailyLogWriter(dir, prefix, 15)
	require.NoError(t, err)
	_, err = writer.Write([]byte("first\n"))
	require.NoError(t, err)
	_, err = writer.Write([]byte("second\n"))
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, prefix+"-"+now.Format("2006-01-02")+".log"))
	require.NoError(t, err)
	assert.Equal(t, "first\nsecond\n", string(content))
	assert.NoFileExists(t, expiredPath)
	assert.FileExists(t, retainedPath)
}

func setLogEnvironment(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("API_LOG_ACCESS_TOKEN", testLogAccessHash)
	t.Setenv("UNISINOS_USP_API_LOG_DIR", dir)
	t.Setenv("UNISINOS_USP_API_LOG_FILE_PREFIX", "app")
}

func performLogRequest(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, path, nil)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	routesSetup().ServeHTTP(recorder, request)
	return recorder
}

func writeLogFixture(t *testing.T, path string, totalLines int) {
	t.Helper()
	lines := make([]string, 0, totalLines)
	for index := 1; index <= totalLines; index++ {
		lines = append(lines, fmt.Sprintf("line %03d", index))
	}
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o664))
}

func countLogLines(content string) int {
	if strings.TrimSpace(content) == "" {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(content), "\n"))
}
