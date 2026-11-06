package controllers

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/services"
	"github.com/gin-gonic/gin"
)

// @BasePath /api

// @Summary Get persisted application log
// @Description Returns a paginated tail of the persisted application log for a date. Requires API_LOG_ACCESS_TOKEN configured on the server; disabled when unset.
// @Tags log
// @Produce plain
// @Param access_hash path string true "Access hash"
// @Param date query string false "Log date (YYYY-MM-DD)"
// @Param limit query int false "Number of lines returned" minimum(1) maximum(1000) default(200)
// @Param offset query int false "Line offset from the most recent chunk" minimum(0) default(0)
// @Success 200 {string} string "Plain text log chunk"
// @Failure 404 {string} string "Not Found or log file not found"
// @Failure 422 {string} string "Invalid date, limit or offset."
// @Router /log/{access_hash} [get]
func FindLog(context *gin.Context) {
	expected := services.LogAccessHash()
	if expected == "" || subtle.ConstantTimeCompare([]byte(context.Param("access_hash")), []byte(expected)) != 1 {
		context.Data(http.StatusNotFound, "text/plain; charset=UTF-8", []byte("Not Found"))
		return
	}

	date, limit, offset, ok := parseLogParameters(context)
	if !ok {
		context.Data(http.StatusUnprocessableEntity, "text/plain; charset=UTF-8", []byte("Invalid date, limit or offset."))
		return
	}

	logPath := services.LogFilePath(date)
	if _, err := os.Stat(logPath); err != nil {
		context.Data(http.StatusNotFound, "text/plain; charset=UTF-8", []byte("Log file not found."))
		return
	}

	content, err := readLastLogLines(logPath, limit, offset)
	if err != nil {
		context.Data(http.StatusNotFound, "text/plain; charset=UTF-8", []byte("Log file not found."))
		return
	}
	context.Data(http.StatusOK, "text/plain; charset=UTF-8", []byte(content))
}

func parseLogParameters(context *gin.Context) (time.Time, int, int, bool) {
	date := time.Now()
	if rawDate := strings.TrimSpace(context.Query("date")); rawDate != "" {
		parsedDate, err := time.ParseInLocation("2006-01-02", rawDate, time.Local)
		if err != nil {
			return time.Time{}, 0, 0, false
		}
		date = parsedDate
	}

	limit, err := strconv.Atoi(strings.TrimSpace(context.DefaultQuery("limit", "200")))
	if err != nil || limit < 1 || limit > 1000 {
		return time.Time{}, 0, 0, false
	}
	offset, err := strconv.Atoi(strings.TrimSpace(context.DefaultQuery("offset", "0")))
	if err != nil || offset < 0 {
		return time.Time{}, 0, 0, false
	}
	return date, limit, offset, true
}

func readLastLogLines(path string, limit int, offset int) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	normalized := strings.TrimRight(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	if normalized == "" {
		return "", nil
	}
	lines := strings.Split(normalized, "\n")
	end := len(lines) - offset
	if end <= 0 {
		return "", nil
	}
	start := end - limit
	if start < 0 {
		start = 0
	}
	return strings.Join(lines[start:end], "\n") + "\n", nil
}
