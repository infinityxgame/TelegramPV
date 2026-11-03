package storage

import (
	"database/sql"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"telegrampv/pkg/logbus"
)

func readConfigValue(db *sql.DB, key string) string {
	var val string
	// 1. Consultar en app_config (esquema oficial de TelegramDL / tgdown)
	err := db.QueryRow("SELECT value FROM app_config WHERE key = ? OR key = ? LIMIT 1", key, "tgdl_"+key).Scan(&val)
	if err == nil && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	// 2. Consultar en config (esquema alternativo o de versiones previas)
	err = db.QueryRow("SELECT value FROM config WHERE key = ? OR key = ? LIMIT 1", key, "tgdl_"+key).Scan(&val)
	if err == nil && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return ""
}

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
		log.Printf("[MIGRATE] TelegramPV ya cuenta con credenciales y sesion activa. No requiere migracion.")
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[MIGRATE] No se pudo obtener el directorio home del usuario: %v", err)
		return
	}

	tgdownDir := filepath.Join(home, ".tgdown")
	if fi, err := os.Stat(tgdownDir); err != nil || !fi.IsDir() {
		log.Printf("[MIGRATE] No existe la carpeta .tgdown en %s", home)
		return
	}

	tgdownDBPath := filepath.Join(tgdownDir, "tgdown.sqlite3")
	tgdownSessionPath := filepath.Join(tgdownDir, "tg_session.json")

	sessionFi, err := os.Stat(tgdownSessionPath)
	if err != nil || sessionFi.Size() == 0 {
		log.Printf("[MIGRATE] No existe el archivo de sesion tg_session.json en .tgdown")
		return
	}

	if _, err := os.Stat(tgdownDBPath); err != nil {
		log.Printf("[MIGRATE] No existe la base de datos tgdown.sqlite3 en .tgdown")
		return
	}

	dsn := tgdownDBPath + "?_pragma=busy_timeout(5000)"
	srcDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Printf("[MIGRATE] Error al abrir la base de datos tgdown.sqlite3: %v", err)
		return
	}
	defer srcDB.Close()

	apiID := readConfigValue(srcDB, "api_id")
	apiHash := readConfigValue(srcDB, "api_hash")
	phone := readConfigValue(srcDB, "phone")
	language := readConfigValue(srcDB, "language")

	if apiID == "" || apiHash == "" {
		log.Printf("[MIGRATE] No se encontraron credenciales de API (api_id/api_hash) en app_config o config de .tgdown")
		return
	}

	log.Printf("[MIGRATE] Importando credenciales de TelegramDL: api_id=%s, phone=%s", apiID, phone)

	if err := copyFile(tgdownSessionPath, targetSessionPath); err != nil {
		logbus.Warn(logbus.CatSystem, "No se pudo copiar la sesion de TelegramDL", err.Error())
		log.Printf("[MIGRATE] Error al copiar archivo de sesion: %v", err)
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

	log.Printf("[MIGRATE] Migracion desde TelegramDL completada con exito.")
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
