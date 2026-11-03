package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCheckAndMigrateFromTelegramDL(t *testing.T) {
	tempHome := t.TempDir()
	origHome := os.Getenv("USERPROFILE")
	if origHome == "" {
		origHome = os.Getenv("HOME")
	}
	os.Setenv("USERPROFILE", tempHome)
	os.Setenv("HOME", tempHome)
	defer func() {
		os.Setenv("USERPROFILE", origHome)
		os.Setenv("HOME", origHome)
	}()

	// Crear estructura simulada de .tgdown
	tgdownDir := filepath.Join(tempHome, ".tgdown")
	if err := os.MkdirAll(tgdownDir, 0700); err != nil {
		t.Fatalf("Failed to create mock .tgdown dir: %v", err)
	}

	// Crear archivo tg_session.json simulado
	mockSessionPath := filepath.Join(tgdownDir, "tg_session.json")
	if err := os.WriteFile(mockSessionPath, []byte(`{"mock": "session_data"}`), 0600); err != nil {
		t.Fatalf("Failed to create mock tg_session.json: %v", err)
	}

	// Crear BD sqlite simulada en .tgdown con tabla app_config (esquema de TelegramDL)
	mockDBPath := filepath.Join(tgdownDir, "tgdown.sqlite3")
	db, err := sql.Open("sqlite", mockDBPath)
	if err != nil {
		t.Fatalf("Failed to create mock tgdown.sqlite3: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE app_config (key TEXT PRIMARY KEY, value TEXT);
		INSERT INTO app_config (key, value) VALUES ('api_id', '987654');
		INSERT INTO app_config (key, value) VALUES ('api_hash', 'abcdef0123456789abcdef0123456789');
		INSERT INTO app_config (key, value) VALUES ('phone', '+1234567890');
		INSERT INTO app_config (key, value) VALUES ('language', 'es');
	`)
	if err != nil {
		t.Fatalf("Failed to populate mock tgdown.sqlite3: %v", err)
	}
	db.Close()

	// Crear DB de destino (TelegramPV) en un directorio temporal
	targetDataDir := filepath.Join(tempHome, ".telegrampv")
	if err := os.MkdirAll(targetDataDir, 0700); err != nil {
		t.Fatalf("Failed to create mock .telegrampv dir: %v", err)
	}

	targetDBPath := filepath.Join(targetDataDir, "telegrampv.sqlite3")
	targetStorage, err := OpenResilient(targetDBPath)
	if err != nil {
		t.Fatalf("Failed to open target storage: %v", err)
	}
	defer targetStorage.Close()

	// Ejecutar migración
	CheckAndMigrateFromTelegramDL(targetStorage, targetDataDir)

	// Verificar resultados
	apiID := targetStorage.GetConfigValue("api_id", "")
	apiHash := targetStorage.GetConfigValue("api_hash", "")
	phone := targetStorage.GetConfigValue("phone", "")
	lang := targetStorage.GetConfigValue("language", "")

	if apiID != "987654" {
		t.Errorf("Expected api_id=987654, got '%s'", apiID)
	}
	if apiHash != "abcdef0123456789abcdef0123456789" {
		t.Errorf("Expected api_hash=abcdef0123456789abcdef0123456789, got '%s'", apiHash)
	}
	if phone != "+1234567890" {
		t.Errorf("Expected phone=+1234567890, got '%s'", phone)
	}
	if lang != "es" {
		t.Errorf("Expected language=es, got '%s'", lang)
	}

	// Verificar que el archivo tg_session.json fue copiado
	targetSessionPath := filepath.Join(targetDataDir, "tg_session.json")
	copiedData, err := os.ReadFile(targetSessionPath)
	if err != nil {
		t.Fatalf("Failed to read copied tg_session.json: %v", err)
	}
	if string(copiedData) != `{"mock": "session_data"}` {
		t.Errorf("Copied session data mismatch: got '%s'", string(copiedData))
	}
}
