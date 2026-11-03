package logbus

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	CatSystem   = "SYSTEM"
	CatTelegram = "TELEGRAM"
	CatAPI      = "API"
	CatServer   = "SERVER"
	CatUpdater  = "UPDATER"

	LevelInfo    = "INFO"
	LevelWarn    = "WARN"
	LevelError   = "ERROR"
	LevelSuccess = "SUCCESS"
)

type LogEntry struct {
	ID        int64  `json:"id"`
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Category  string `json:"category"`
	Message   string `json:"message"`
	Details   string `json:"details"`
}

var (
	mu          sync.RWMutex
	logEntries  []LogEntry
	nextID      int64 = 1
	logFile     *os.File
	listeners   []func(LogEntry)
	attached    bool
)

func Init(dataDir string) error {
	mu.Lock()
	defer mu.Unlock()

	logsDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logsDir, 0700); err != nil {
		return err
	}

	fileName := fmt.Sprintf("log_%s.log", time.Now().Format("02-01-06"))
	filePath := filepath.Join(logsDir, fileName)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	logFile = f
	return nil
}

func Attach() {
	mu.Lock()
	attached = true
	mu.Unlock()
}

func FilePath() string {
	mu.RLock()
	defer mu.RUnlock()
	if logFile != nil {
		return logFile.Name()
	}
	return ""
}

func AddListener(fn func(LogEntry)) {
	mu.Lock()
	defer mu.Unlock()
	listeners = append(listeners, fn)
}

func logInternal(level, category, message, details string) {
	mu.Lock()
	entry := LogEntry{
		ID:        nextID,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Level:     level,
		Category:  category,
		Message:   message,
		Details:   details,
	}
	nextID++
	logEntries = append(logEntries, entry)
	if len(logEntries) > 1000 {
		logEntries = logEntries[len(logEntries)-1000:]
	}

	if logFile != nil {
		line := fmt.Sprintf("[%s] [%s] [%s] %s %s\n", entry.Timestamp, entry.Level, entry.Category, entry.Message, entry.Details)
		_, _ = logFile.WriteString(line)
	}

	listenersCopy := make([]func(LogEntry), len(listeners))
	copy(listenersCopy, listeners)
	mu.Unlock()

	log.Printf("[%s] [%s] %s %s", level, category, message, details)

	for _, listener := range listenersCopy {
		listener(entry)
	}
}

func Info(category, message string, details ...string) {
	det := ""
	if len(details) > 0 {
		det = details[0]
	}
	logInternal(LevelInfo, category, message, det)
}

func Warn(category, message string, details ...string) {
	det := ""
	if len(details) > 0 {
		det = details[0]
	}
	logInternal(LevelWarn, category, message, det)
}

func Error(category, message string, details ...string) {
	det := ""
	if len(details) > 0 {
		det = details[0]
	}
	logInternal(LevelError, category, message, det)
}

func Success(category, message string, details ...string) {
	det := ""
	if len(details) > 0 {
		det = details[0]
	}
	logInternal(LevelSuccess, category, message, det)
}

func GetEntries(sinceID int64) ([]LogEntry, int64) {
	mu.RLock()
	defer mu.RUnlock()

	var result []LogEntry
	maxID := sinceID
	for _, entry := range logEntries {
		if entry.ID > sinceID {
			result = append(result, entry)
			if entry.ID > maxID {
				maxID = entry.ID
			}
		}
	}
	return result, maxID
}
