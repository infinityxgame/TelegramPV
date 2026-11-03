package config

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	AppVersion = "1.0.2"
	GithubRepo = "infinityxgame/TelegramPV"

	DefaultBindHost = "127.0.0.1"
	DefaultBindPort = 8766
)

type Config struct {
	ColorID       *int   `json:"color_id"`
	LoaderColorID *int   `json:"loader_color_id"`
	Language      string `json:"language"`
	ApiID         string `json:"api_id,omitempty"`
	ApiHash       string `json:"api_hash,omitempty"`
	Phone         string `json:"phone,omitempty"`
}

func DefaultConfig() Config {
	defaultColor := 5
	return Config{
		ColorID:       &defaultColor,
		LoaderColorID: &defaultColor,
		Language:      "",
	}
}

var (
	DataDir     string
	UserEnvPath string
	BaseDir     string
	initDirOnce sync.Once

	serverBindHost = DefaultBindHost
	serverBindPort = DefaultBindPort
	bindMu         sync.RWMutex
)

func InitPaths() {
	initDirOnce.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		DataDir = filepath.Join(home, ".telegrampv")
		_ = os.MkdirAll(DataDir, 0700)

		UserEnvPath = filepath.Join(DataDir, ".env")

		execPath, err := os.Executable()
		if err == nil {
			BaseDir = filepath.Dir(execPath)
		} else {
			BaseDir = "."
		}

		SecureDataDir()
	})
}

func SecureDataDir(extraPaths ...string) {
	if err := os.Chmod(DataDir, 0700); err != nil {
		log.Printf("[CONFIG] Could not secure data dir mode: %v", err)
	}

	sensitiveNames := []string{
		"tg_session.json",
		"telegrampv.sqlite3",
		"telegrampv.sqlite3-wal",
		"telegrampv.sqlite3-shm",
		".env",
	}

	for _, name := range sensitiveNames {
		p := filepath.Join(DataDir, name)
		if _, err := os.Stat(p); err == nil {
			_ = os.Chmod(p, 0600)
		}
	}

	for _, p := range extraPaths {
		if p != "" {
			if _, err := os.Stat(p); err == nil {
				_ = os.Chmod(p, 0600)
			}
		}
	}
}

func SetServerBinding(host string, port int) {
	bindMu.Lock()
	defer bindMu.Unlock()
	if strings.TrimSpace(host) != "" {
		serverBindHost = host
	}
	if port > 0 {
		serverBindPort = port
	}
}

func GetServerHost() string {
	bindMu.RLock()
	defer bindMu.RUnlock()
	return serverBindHost
}

func GetServerPort() int {
	bindMu.RLock()
	defer bindMu.RUnlock()
	return serverBindPort
}

func EsLoopback(host string) bool {
	h := strings.TrimSpace(host)
	if h == "" {
		return false
	}
	if hSinPuerto, _, err := net.SplitHostPort(h); err == nil {
		h = strings.TrimSpace(hSinPuerto)
	}
	if strings.EqualFold(h, "localhost") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}
