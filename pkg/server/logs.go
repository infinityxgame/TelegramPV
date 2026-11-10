package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"telegrampv/pkg/config"
	"telegrampv/pkg/i18n"
	"telegrampv/pkg/logbus"
)

// handleLogs devuelve las entradas del registro de actividad en formato JSON.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, i18n.T("server.methodNotAllowed"))
		return
	}

	var since int64 = -1
	if raw := strings.TrimSpace(r.URL.Query().Get("since")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			since = parsed
		}
	}

	limit := 800
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	entries, lastID := logbus.Since(since, limit)
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"entries": entries,
		"logs":    entries,
		"last_id": lastID,
		"file":    logbus.FilePath(),
	})
}

// handleLogsClear vacía las entradas en memoria del registro.
func (s *Server) handleLogsClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		jsonError(w, http.StatusMethodNotAllowed, i18n.T("server.methodNotAllowed"))
		return
	}
	logbus.Clear()
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"last_id": logbus.LastID(),
	})
}

// handleLogsExport entrega el registro completo en memoria como archivo descargable.
func (s *Server) handleLogsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, i18n.T("server.methodNotAllowed"))
		return
	}
	name := fmt.Sprintf("telegrampv-%s.log", time.Now().Format("2006-01-02_15-04-05"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(logbus.ExportText()))
}

// logConfigChanges compara la configuración anterior con la nueva y registra los cambios.
func logConfigChanges(before, after config.Config) {
	changes := make([]string, 0, 6)

	if !sameIntPtr(before.ColorID, after.ColorID) {
		changes = append(changes, i18n.T("settings.changeColor"))
	}
	if !sameIntPtr(before.LoaderColorID, after.LoaderColorID) {
		changes = append(changes, i18n.T("settings.changeLoaderColor"))
	}
	if before.Language != after.Language && after.Language != "" {
		beforeLang := before.Language
		if beforeLang == "" {
			beforeLang = "auto"
		}
		changes = append(changes, fmt.Sprintf(i18n.T("settings.changeLanguage"), beforeLang, after.Language))
	}
	if (before.ApiID != after.ApiID && after.ApiID != "") || (before.ApiHash != after.ApiHash && after.ApiHash != "") {
		changes = append(changes, i18n.T("settings.changeApiCreds"))
	}
	if before.Phone != after.Phone && after.Phone != "" {
		changes = append(changes, fmt.Sprintf(i18n.T("settings.changePhone"), after.Phone))
	}

	if len(changes) > 0 {
		logbus.Info(logbus.CatSettings, i18n.T("settings.saved"), strings.Join(changes, " · "))
	}
}

func sameIntPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
