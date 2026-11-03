package storage

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"telegrampv/pkg/logbus"
)

// CheckAndMigrateFromTelegramDL comprueba si existe la carpeta de TelegramDL (.tgdown)
// con una base de datos y sesión activa de Telegram, y las copia a TelegramPV (.telegrampv)
// si este último todavía no tiene una sesión y credenciales configuradas.
func CheckAndMigrateFromTelegramDL(targetStorage *Storage, targetDataDir string) {
	targetApiID := targetStorage.GetConfigValue("api_id", "")
	targetApiHash := targetStorage.GetConfigValue("api_hash", "")
	targetSessionPath := filepath.Join(targetDataDir, "tg_session.json")

	hasTargetSession := false
	if fi, err := os.Stat(targetSessionPath); err == nil && fi.Size() > 0 {
		hasTargetSession = true
	}

	if targetApiID != "" && targetApiHash != "" && hasTargetSession {
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	tgdownDir := filepath.Join(home, ".tgdown")
	if fi, err := os.Stat(tgdownDir); err != nil || !fi.IsDir() {
		return
	}

	tgdownDBPath := filepath.Join(tgdownDir, "tgdown.sqlite3")
	tgdownSessionPath := filepath.Join(tgdownDir, "tg_session.json")

	sessionFi, err := os.Stat(tgdownSessionPath)
	if err != nil || sessionFi.Size() == 0 {
		return
	}

	if _, err := os.Stat(tgdownDBPath); err != nil {
		return
	}

	srcDB, err := sql.Open("sqlite", tgdownDBPath)
	if err != nil {
		return
	}
	defer srcDB.Close()

	var apiID, apiHash, phone, language string
	_ = srcDB.QueryRow("SELECT value FROM config WHERE key = 'api_id'").Scan(&apiID)
	_ = srcDB.QueryRow("SELECT value FROM config WHERE key = 'api_hash'").Scan(&apiHash)
	_ = srcDB.QueryRow("SELECT value FROM config WHERE key = 'phone'").Scan(&phone)
	_ = srcDB.QueryRow("SELECT value FROM config WHERE key = 'language'").Scan(&language)

	apiID = strings.TrimSpace(apiID)
	apiHash = strings.TrimSpace(apiHash)

	if apiID == "" || apiHash == "" {
		return
	}

	if err := copyFile(tgdownSessionPath, targetSessionPath); err != nil {
		logbus.Warn(logbus.CatSystem, "No se pudo copiar la sesion de TelegramDL: "+err.Error())
		return
	}

	_ = targetStorage.SaveConfigKey("api_id", apiID)
	_ = targetStorage.SaveConfigKey("api_hash", apiHash)
	if phone != "" {
		_ = targetStorage.SaveConfigKey("phone", phone)
	}
	if language != "" {
		_ = targetStorage.SaveConfigKey("language", language)
	}

	logbus.Success(logbus.CatSystem, "Sesion y credenciales importadas exitosamente desde TelegramDL", "")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
