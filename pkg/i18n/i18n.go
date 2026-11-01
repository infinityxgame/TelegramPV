package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	currentLang = "en"
	mu          sync.RWMutex
	messages    = map[string]map[string]string{
		"en": {
			"system.appStarted":        "TelegramPV v%s started",
			"system.dbOpenFatal":       "Fatal error opening SQLite database: %s",
			"system.logFileActive":     "Log file active at: %s",
			"system.logFileError":      "Could not initialize log file: %s",
			"telegram.notConnected":    "Telegram client is not connected",
			"telegram.notConnectedShort": "Not connected to Telegram",
			"telegram.notConfigured":   "API credentials have not been configured",
			"telegram.connecting":      "Connecting to Telegram servers...",
			"telegram.connected":       "Successfully connected to Telegram",
			"telegram.sessionActive":   "Active session for user: %s %s (@%s, ID: %d)",
			"telegram.waitingAuth":     "Waiting for user authentication...",
			"telegram.runError":        "Telegram client error: %v",
			"telegram.invalidAPIID":    "Invalid API ID, must be a number: %w",
			"telegram.connectError":    "Connection error: %v",
			"telegram.sendCodeError":   "Error sending code: %v",
			"telegram.unexpectedResponse": "Unexpected response from Telegram",
			"telegram.verifyCodeError": "Error verifying code: %v",
			"telegram.2faRequired":     "2FA password required",
			"telegram.2faError":        "Error authenticating with 2FA password: %v",
			"telegram.fetchDialogsError": "Error fetching chats from Telegram: %v",
			"telegram.leaveChannelError": "Error leaving channel/group: %v",
			"telegram.deleteHistoryError": "Error deleting chat history: %v",
		},
		"es": {
			"system.appStarted":        "TelegramPV v%s iniciado",
			"system.dbOpenFatal":       "Error fatal abriendo la base de datos SQLite: %s",
			"system.logFileActive":     "Archivo de registro activo en: %s",
			"system.logFileError":      "No se pudo inicializar el archivo de registros: %s",
			"telegram.notConnected":    "El cliente de Telegram no está conectado",
			"telegram.notConnectedShort": "No conectado a Telegram",
			"telegram.notConfigured":   "No se han configurado las credenciales de API",
			"telegram.connecting":      "Conectando a los servidores de Telegram...",
			"telegram.connected":       "Conectado exitosamente a Telegram",
			"telegram.sessionActive":   "Sesión activa para el usuario: %s %s (@%s, ID: %d)",
			"telegram.waitingAuth":     "Esperando autenticación del usuario...",
			"telegram.runError":        "Error en el cliente de Telegram: %v",
			"telegram.invalidAPIID":    "API ID inválido, debe ser un número: %w",
			"telegram.connectError":    "Error de conexión: %v",
			"telegram.sendCodeError":   "Error al enviar código: %v",
			"telegram.unexpectedResponse": "Respuesta inesperada de Telegram",
			"telegram.verifyCodeError": "Error al verificar código: %v",
			"telegram.2faRequired":     "Se requiere contraseña 2FA",
			"telegram.2faError":        "Error al autenticar con contraseña 2FA: %v",
			"telegram.fetchDialogsError": "Error al obtener chats de Telegram: %v",
			"telegram.leaveChannelError": "Error al salir del canal/grupo: %v",
			"telegram.deleteHistoryError": "Error al eliminar el historial del chat: %v",
		},
	}
)

// DetectSystemLanguage detecta el idioma del sistema operativo.
func DetectSystemLanguage() string {
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = os.Getenv("LC_ALL")
	}
	if lang == "" {
		lang = os.Getenv("USERLEVEL")
	}
	lang = strings.ToLower(lang)
	if strings.HasPrefix(lang, "es") {
		return "es"
	}
	return "en"
}

// SetLanguage cambia el idioma global del backend.
func SetLanguage(lang string) {
	mu.Lock()
	defer mu.Unlock()
	if lang == "es" || lang == "en" {
		currentLang = lang
	}
}

// CurrentLanguage devuelve el idioma actual.
func CurrentLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// T traduce una clave con argumentos opcionales.
func T(key string, args ...interface{}) string {
	mu.RLock()
	lang := currentLang
	dict := messages[lang]
	msg, exists := dict[key]
	if !exists {
		// Fallback a inglés si no existe en el idioma actual
		if engDict, ok := messages["en"]; ok {
			msg, exists = engDict[key]
		}
	}
	mu.RUnlock()

	if !exists {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}
