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
			"system.appStarted":            "TelegramPV v%s started",
			"system.dbOpenFatal":           "Fatal error opening SQLite database: %s",
			"system.logFileActive":         "Log file active at: %s",
			"system.logFileError":          "Could not initialize log file: %s",
			"telegram.notConnected":        "Telegram client is not connected. Please check your API ID, API Hash or internet connection.",
			"telegram.notConnectedShort":   "Not connected to Telegram",
			"telegram.notConfigured":       "API credentials have not been configured",
			"telegram.connecting":          "Connecting to Telegram servers...",
			"telegram.connected":           "Successfully connected to Telegram",
			"telegram.sessionActive":       "Active session for user: %s %s (@%s, ID: %d)",
			"telegram.waitingAuth":         "Waiting for user authentication...",
			"telegram.runError":            "Telegram client error: %v",
			"telegram.invalidAPIID":        "Invalid API ID, must be a number: %w",
			"telegram.connectError":        "%v",
			"telegram.sendCodeError":       "Error sending code: %v",
			"telegram.unexpectedResponse":  "Unexpected response from Telegram",
			"telegram.verifyCodeError":     "Error verifying code: %v",
			"telegram.2faRequired":         "2FA password required",
			"telegram.2faError":            "Error authenticating with 2FA password: %v",
			"telegram.fetchDialogsError":   "Error fetching chats from Telegram: %v",
			"telegram.leaveChannelError":   "Error leaving channel/group: %v",
			"telegram.deleteHistoryError": "Error deleting chat history: %v",

			// Updater
			"updater.rateLimited":           "GitHub rate-limited the update checks",
			"updater.rateLimitedDetail":     "The hourly request limit was reached; not asking again for %s",
			"updater.checkPaused":           "Update check paused",
			"updater.checkPausedDetail":     "GitHub limited the requests; retrying in %s",
			"updater.checking":              "Checking for updates of %s",
			"updater.checkingDetail":        "Installed version: v%s",
			"updater.checkError":            "Could not fetch the latest published release",
			"updater.noReleases":            "There is no published release yet",
			"updater.noAsset":               "No file compatible with this system was found",
			"updater.noAssetDetail":         "Published version: %s (%d files)",
			"updater.latestRelease":         "Latest published release: %s",
			"updater.latestReleaseDetail":   "File: %s",
			"updater.upToDate":              "TelegramPV is up to date (v%s)",
			"updater.stillAvailable":        "Version %s is still available",
			"updater.newVersion":            "New version available: %s!",
			"updater.newVersionDetail":      "You have v%s · You can update from Settings",
			"updater.verifyRejected":        "Update rejected: verification failed",
			"updater.applyError":            "Could not apply the update",
			"updater.applying":              "Applying update over: %s",
			"updater.applied":               "Update applied",
			"updater.cleanupNextBoot":       "The old version will be removed on the next start: %v",
			"updater.relaunching":           "Launching new version: %s",
			"updater.relaunchError":         "Could not relaunch the application: %v",
			"updater.verified":              "Update integrity verified",
			"updater.oldFileFound":          "Old version file found: %s. Removing...",
			"updater.oldFileError":          "Could not remove the old version %s: %v",
			"updater.oldFileRemoved":        "Old version removed: %s",
			"updater.oldBundleFound":        "Old version bundle found: %s. Removing...",
			"updater.oldBundleError":        "Could not remove %s: %v",
			"updater.installStartError":     "Could not start the update",
			"updater.installing":            "Installing update %s",
			"updater.installingDetail":      "Current version: v%s",
			"updater.downloadFailedPrefix":  "error: download failed: ",
			"updater.unzipFailedPrefix":     "error: failed to unzip file: ",
			"updater.untarFailedPrefix":     "error: failed to untar file: ",
			"updater.exeNotFound":            "error: executable not found in download",
			"updater.applyFailedPrefix":     "error: failed to apply update: ",
			"updater.repoMisconfigured":     "Repository misconfigured: %s",
			"updater.githubRejected":        "GitHub rejected request with status %d",
			"updater.githubResponseCode":    "GitHub returned status code %d",
			"updater.githubReadError":        "Error reading GitHub response: %v",
			"updater.updateInfoInvalid":     "Update info is invalid",
			"updater.tempDirError":          "Could not create temporary directory",
			"updater.exePathError":          "Could not determine executable path",
			"updater.readNewVersionError":   "Could not read new version: %v",
			"updater.writeError":            "Could not write to %s: %v",
			"updater.copyError":             "Could not copy binary: %v",
			"updater.closeError":            "Could not close file: %v",
			"updater.chmodError":            "Could not set permissions: %v",
			"updater.moveOldError":          "Could not rename current binary %s: %v",
			"updater.rollbackFailed":        "Rollback failed: %v, %s: %v",
			"updater.putNewError":           "Could not place new binary: %v",
			"updater.githubStatusCode":      "GitHub download status code %d",
			"updater.downloadTimeout":       "Download timed out after %v",
			"updater.downloadIncomplete":    "Downloaded incomplete file (%d of %d bytes)",
			"updater.signatureError":        "Error calculating checksum: %v",
			"updater.signatureMismatch":     "Checksum mismatch: expected %s, got %s",
			"updater.checksumDownloadError": "Could not download %s: %v",
			"updater.checksumNotPublished":  "%s not published (status %d)",
			"updater.checksumReadError":      "Error reading %s: %v",
			"updater.checksumMissingEntry":  "%s missing entry for %s",
			"updater.invalidDownloadURL":     "Invalid download URL: %v",
			"updater.invalidDownloadURLText": "Invalid download URL: %s",
			"updater.darwinTranslocation":   "Cannot update application running from AppTranslocation",
			"updater.darwinTempDirError":    "Could not create temporary directory: %v",
			"updater.darwinNoAppBundle":      "No .app bundle found in download",
			"updater.darwinReplaceError":     "Could not set aside current bundle %s: %v",
			"updater.darwinRollbackFailed":   "Rollback failed: %v, %v",
			"updater.darwinCopyError":        "Could not copy bundle: %v",
			"updater.darwinExtractError":     "Could not extract macOS archive: %v",
			"updater.darwinUnsupportedFormat": "Unsupported archive format: %s",
			"server.noUpdatesAvailable":     "No updates available",
			"server.startingInstall":        "Starting install",
			"server.versionRequired":        "Version required",
			"cli.checkError":                "Error checking updates: %v\n",
			"cli.alreadyUpdated":            "Application is already updated (v%s).\n",
			"cli.updating":                  "Updating from v%s to %s with %s...\n",
			"cli.installError":              "Error starting update: %v\n",
			"cli.progressState":             "Status: %s (%d%%)\n",
			"cli.usageUpdate":               "  TelegramPV.exe --update     Check and install latest update",
		},
		"es": {
			"system.appStarted":            "TelegramPV v%s iniciado",
			"system.dbOpenFatal":           "Error fatal abriendo la base de datos SQLite: %s",
			"system.logFileActive":         "Archivo de registro activo en: %s",
			"system.logFileError":          "No se pudo inicializar el archivo de registros: %s",
			"telegram.notConnected":        "El cliente de Telegram no está conectado. Comprueba tu API ID, API Hash y conexión a internet.",
			"telegram.notConnectedShort":   "No conectado a Telegram",
			"telegram.notConfigured":       "No se han configurado las credenciales de API",
			"telegram.connecting":          "Conectando a los servidores de Telegram...",
			"telegram.connected":           "Conectado exitosamente a Telegram",
			"telegram.sessionActive":       "Sesión activa para el usuario: %s %s (@%s, ID: %d)",
			"telegram.waitingAuth":         "Esperando autenticación del usuario...",
			"telegram.runError":            "Error en el cliente de Telegram: %v",
			"telegram.invalidAPIID":        "API ID inválido, debe ser un número: %w",
			"telegram.connectError":        "%v",
			"telegram.sendCodeError":       "Error al enviar código: %v",
			"telegram.unexpectedResponse":  "Respuesta inesperada de Telegram",
			"telegram.verifyCodeError":     "Error al verificar código: %v",
			"telegram.2faRequired":         "Se requiere contraseña 2FA",
			"telegram.2faError":            "Error al autenticar con contraseña 2FA: %v",
			"telegram.fetchDialogsError":   "Error al obtener chats de Telegram: %v",
			"telegram.leaveChannelError":   "Error al salir del canal/grupo: %v",
			"telegram.deleteHistoryError": "Error al eliminar el historial del chat: %v",

			// Updater
			"updater.rateLimited":           "GitHub limitó las comprobaciones de actualización",
			"updater.rateLimitedDetail":     "Se alcanzó el máximo de peticiones por hora; no se volverá a preguntar hasta dentro de %s",
			"updater.checkPaused":           "Comprobación de actualizaciones en pausa",
			"updater.checkPausedDetail":     "GitHub limitó las peticiones; se reintentará en %s",
			"updater.checking":              "Comprobando actualizaciones de %s",
			"updater.checkingDetail":        "Versión instalada: v%s",
			"updater.checkError":            "No se pudo consultar la última versión publicada",
			"updater.noReleases":            "Todavía no hay ninguna versión publicada",
			"updater.noAsset":               "No se encontró ningún archivo compatible con este sistema",
			"updater.noAssetDetail":         "Versión publicada: %s (%d archivos)",
			"updater.latestRelease":         "Última versión publicada: %s",
			"updater.latestReleaseDetail":   "Archivo: %s",
			"updater.upToDate":              "TelegramPV está al día (v%s)",
			"updater.stillAvailable":        "Sigue disponible la versión %s",
			"updater.newVersion":            "¡Nueva versión disponible: %s!",
			"updater.newVersionDetail":      "Tienes la v%s · Puedes actualizar desde Ajustes",
			"updater.verifyRejected":        "Actualización rechazada por no superar la verificación",
			"updater.applyError":            "No se pudo aplicar la actualización",
			"updater.applying":              "Aplicando actualización sobre: %s",
			"updater.applied":               "Actualización aplicada",
			"updater.cleanupNextBoot":       "La versión anterior se eliminará en el próximo arranque: %v",
			"updater.relaunching":           "Lanzando nueva versión: %s",
			"updater.relaunchError":         "No se pudo relanzar la aplicación: %v",
			"updater.verified":              "Integridad de la actualización verificada",
			"updater.oldFileFound":          "Detectado archivo de versión antigua: %s. Eliminando...",
			"updater.oldFileError":          "No se pudo eliminar la versión antigua %s: %v",
			"updater.oldFileRemoved":        "Versión antigua eliminada: %s",
			"updater.oldBundleFound":        "Detectado bundle de versión antigua: %s. Eliminando...",
			"updater.oldBundleError":        "No se pudo eliminar %s: %v",
			"updater.installStartError":     "No se pudo iniciar la actualización",
			"updater.installing":            "Instalando actualización %s",
			"updater.installingDetail":      "Versión actual: v%s",
			"updater.downloadFailedPrefix":  "error: descarga fallida: ",
			"updater.unzipFailedPrefix":     "error: fallo al descomprimir zip: ",
			"updater.untarFailedPrefix":     "error: fallo al descomprimir tar.gz: ",
			"updater.exeNotFound":            "error: no se encontró el ejecutable en la descarga",
			"updater.applyFailedPrefix":     "error: fallo al aplicar actualización: ",
			"updater.repoMisconfigured":     "Repositorio mal configurado: %s",
			"updater.githubRejected":        "GitHub rechazó la petición con código %d",
			"updater.githubResponseCode":    "GitHub devolvió código %d",
			"updater.githubReadError":        "Error leyendo respuesta de GitHub: %v",
			"updater.updateInfoInvalid":     "Información de actualización no válida",
			"updater.tempDirError":          "No se pudo crear directorio temporal",
			"updater.exePathError":          "No se pudo obtener la ruta del ejecutable",
			"updater.readNewVersionError":   "Error leyendo nueva versión: %v",
			"updater.writeError":            "Error escribiendo en %s: %v",
			"updater.copyError":             "Error copiando binario: %v",
			"updater.closeError":            "Error cerrando archivo: %v",
			"updater.chmodError":            "Error cambiando permisos: %v",
			"updater.moveOldError":          "Error cambiando nombre del ejecutable actual %s: %v",
			"updater.rollbackFailed":        "Falló la restauración: %v, %s: %v",
			"updater.putNewError":           "Error colocando nuevo binario: %v",
			"updater.githubStatusCode":      "Código de estado de descarga de GitHub %d",
			"updater.downloadTimeout":       "Se agotó el tiempo de descarga tras %v",
			"updater.downloadIncomplete":    "Descarga incompleta (%d de %d bytes)",
			"updater.signatureError":        "Error calculando suma de verificación: %v",
			"updater.signatureMismatch":     "Suma de verificación no coincide: se esperaba %s, se obtuvo %s",
			"updater.checksumDownloadError": "No se pudo descargar %s: %v",
			"updater.checksumNotPublished":  "%s no publicado (código %d)",
			"updater.checksumReadError":      "Error leyendo %s: %v",
			"updater.checksumMissingEntry":  "%s no contiene entrada para %s",
			"updater.invalidDownloadURL":     "URL de descarga no válida: %v",
			"updater.invalidDownloadURLText": "URL de descarga no válida: %s",
			"updater.darwinTranslocation":   "No se puede actualizar ejecutando desde AppTranslocation",
			"updater.darwinTempDirError":    "No se pudo crear directorio temporal: %v",
			"updater.darwinNoAppBundle":      "No se encontró bundle .app en la descarga",
			"updater.darwinReplaceError":     "No se pudo apartar bundle actual %s: %v",
			"updater.darwinRollbackFailed":   "Falló la restauración: %v, %v",
			"updater.darwinCopyError":        "Error copiando bundle: %v",
			"updater.darwinExtractError":     "Error extrayendo archivo de macOS: %v",
			"updater.darwinUnsupportedFormat": "Formato de archivo no soportado: %s",
			"server.noUpdatesAvailable":     "No hay actualizaciones disponibles",
			"server.startingInstall":        "Iniciando instalación",
			"server.versionRequired":        "Versión requerida",
			"cli.checkError":                "Error al comprobar actualizaciones: %v\n",
			"cli.alreadyUpdated":            "La aplicación ya está actualizada (v%s).\n",
			"cli.updating":                  "Actualizando de v%s a %s con %s...\n",
			"cli.installError":              "Error al iniciar la actualización: %v\n",
			"cli.progressState":             "Estado: %s (%d%%)\n",
			"cli.usageUpdate":               "  TelegramPV.exe --update     Buscar e instalar la última actualización",
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
