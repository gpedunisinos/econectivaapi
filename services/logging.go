package services

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	logrus "github.com/sirupsen/logrus"
)

const (
	DefaultLogDir           = "storage/logs"
	DefaultLogFilePrefix    = "app"
	DefaultLogRetentionDays = 15
)

var (
	loggingConfigured = false
	loggingMu         sync.Mutex
	originalStdout    = os.Stdout
	originalStderr    = os.Stderr
)

type dailyLogWriter struct {
	mu            sync.Mutex
	dir           string
	prefix        string
	retentionDays int
	currentDate   string
	file          *os.File
}

func SetupLogging() error {
	loggingMu.Lock()
	defer loggingMu.Unlock()

	if loggingConfigured {
		return nil
	}

	writer, err := NewDailyLogWriter(LogDir(), LogFilePrefix(), DefaultLogRetentionDays)
	if err != nil {
		return err
	}

	stdoutWriter := io.MultiWriter(originalStdout, writer)
	stderrWriter := io.MultiWriter(originalStderr, writer)

	log.SetOutput(stdoutWriter)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	logrus.SetOutput(stdoutWriter)
	UseJSONLogFormat()
	gin.DefaultWriter = stdoutWriter
	gin.DefaultErrorWriter = stderrWriter

	if err := redirectLogStream(&os.Stdout, stdoutWriter); err != nil {
		return err
	}
	if err := redirectLogStream(&os.Stderr, stderrWriter); err != nil {
		return err
	}

	loggingConfigured = true
	return nil
}

func NewDailyLogWriter(dir string, prefix string, retentionDays int) (io.Writer, error) {
	writer := &dailyLogWriter{dir: dir, prefix: prefix, retentionDays: retentionDays}
	if err := writer.prepare(time.Now()); err != nil {
		return nil, err
	}
	return writer, nil
}

// LogAccessHash returns the deployment-specific log credential.
// An empty value disables remote access to persisted logs.
func LogAccessHash() string {
	return strings.TrimSpace(os.Getenv("API_LOG_ACCESS_TOKEN"))
}

func LogDir() string {
	if value := strings.TrimSpace(os.Getenv("UNISINOS_USP_API_LOG_DIR")); value != "" {
		return value
	}
	return DefaultLogDir
}

func LogFilePrefix() string {
	if value := strings.TrimSpace(os.Getenv("UNISINOS_USP_API_LOG_FILE_PREFIX")); value != "" {
		return value
	}
	return DefaultLogFilePrefix
}

func LogFilePath(date time.Time) string {
	return filepath.Join(LogDir(), LogFilePrefix()+"-"+date.Format("2006-01-02")+".log")
}

func (writer *dailyLogWriter) Write(content []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if err := writer.prepare(time.Now()); err != nil {
		return 0, err
	}
	return writer.file.Write(content)
}

func (writer *dailyLogWriter) prepare(now time.Time) error {
	date := now.Format("2006-01-02")
	if writer.file != nil && writer.currentDate == date {
		return nil
	}

	if writer.file != nil {
		_ = writer.file.Close()
	}
	if err := os.MkdirAll(writer.dir, 0o775); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Join(writer.dir, writer.prefix+"-"+date+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o664)
	if err != nil {
		return err
	}
	writer.file = file
	writer.currentDate = date
	writer.removeExpired(now)
	return nil
}

func (writer *dailyLogWriter) removeExpired(now time.Time) {
	entries, err := os.ReadDir(writer.dir)
	if err != nil {
		return
	}

	oldestDate := startOfLogDay(now).AddDate(0, 0, -(writer.retentionDays - 1))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		dateText := strings.TrimSuffix(strings.TrimPrefix(name, writer.prefix+"-"), ".log")
		if name != writer.prefix+"-"+dateText+".log" {
			continue
		}
		fileDate, err := time.ParseInLocation("2006-01-02", dateText, now.Location())
		if err == nil && fileDate.Before(oldestDate) {
			_ = os.Remove(filepath.Join(writer.dir, name))
		}
	}
}

func startOfLogDay(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}

func redirectLogStream(target **os.File, destination io.Writer) error {
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	*target = writer
	go func() {
		_, _ = io.Copy(destination, reader)
	}()
	return nil
}
