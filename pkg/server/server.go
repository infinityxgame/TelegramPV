package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"telegrampv/pkg/config"
	"telegrampv/pkg/i18n"
	"telegrampv/pkg/logbus"
	"telegrampv/pkg/storage"
	"telegrampv/pkg/telegram"
	"telegrampv/pkg/updater"
)

type Server struct {
	clientMgr *telegram.ClientManager
	storage   *storage.Storage
	updater   *updater.AppUpdater
	assets    fs.FS
	mux       *http.ServeMux
	httpSrv   *http.Server
	latestRel *updater.ReleaseInfo
	mu        sync.Mutex
}

func NewServer(cm *telegram.ClientManager, st *storage.Storage, up *updater.AppUpdater, assets fs.FS) *Server {
	return &Server{
		clientMgr: cm,
		storage:   st,
		updater:   up,
		assets:    assets,
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-TGDL-Request")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start(host string, port int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpSrv != nil {
		_ = s.httpSrv.Close()
	}

	mux := http.NewServeMux()
	s.mux = mux

	// API Routes
	mux.HandleFunc("/api/auth/status", s.handleAuthStatus)
	mux.HandleFunc("/api/auth/has-credentials", s.handleHasCredentials)
	mux.HandleFunc("/api/auth/credentials", s.handleSaveCredentials)
	mux.HandleFunc("/api/auth/send-code", s.handleSendCode)
	mux.HandleFunc("/api/auth/sign-in", s.handleSignIn)
	mux.HandleFunc("/api/auth/verify-code", s.handleSignIn)
	mux.HandleFunc("/api/auth/2fa", s.handleTwoFactor)
	mux.HandleFunc("/api/auth/verify-2fa", s.handleTwoFactor)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)

	mux.HandleFunc("/api/chats", s.handleGetChats)
	mux.HandleFunc("/api/chats/sync", s.handleSyncChats)
	mux.HandleFunc("/api/chats/leave", s.handleLeaveChat)
	mux.HandleFunc("/api/chats/export", s.handleExportChats)
	mux.HandleFunc("/api/chats/open", s.handleOpenTelegram)

	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/logs/clear", s.handleLogsClear)
	mux.HandleFunc("/api/logs/export", s.handleLogsExport)

	// Updates (singular /update/ y plural /updates/)
	mux.HandleFunc("/api/update/check", s.handleCheckUpdate)
	mux.HandleFunc("/api/updates/check", s.handleCheckUpdate)
	mux.HandleFunc("/api/update/progress", s.handleUpdateProgress)
	mux.HandleFunc("/api/updates/progress", s.handleUpdateProgress)
	mux.HandleFunc("/api/update/install", s.handleInstallUpdate)
	mux.HandleFunc("/api/updates/install", s.handleInstallUpdate)
	mux.HandleFunc("/api/update/postpone", s.handlePostponeUpdate)

	// Static Files Handler
	if s.assets != nil {
		fileServer := http.FileServer(http.FS(s.assets))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}

			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}

			_, err := s.assets.Open(path)
			if err != nil {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return "", err
	}

	realPort := listener.Addr().(*net.TCPAddr).Port
	actualAddr := fmt.Sprintf("http://%s:%d", host, realPort)
	config.SetServerBinding(host, realPort)

	s.httpSrv = &http.Server{
		Handler: corsMiddleware(mux),
	}

	go func() {
		if err := s.httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[SERVER] Server error: %v", err)
		}
	}()

	// Comprobación de actualización al iniciar
	if s.updater != nil {
		go func() {
			time.Sleep(1 * time.Second)
			rel, _, err := s.updater.CheckForUpdate()
			if err == nil && rel != nil {
				s.mu.Lock()
				s.latestRel = rel
				s.mu.Unlock()
			}
		}()
	}

	logbus.Info(logbus.CatSystem, fmt.Sprintf("Server listening on %s", actualAddr))
	return actualAddr, nil
}

func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2)
		defer cancel()
		_ = s.httpSrv.Shutdown(ctx)
		s.httpSrv = nil
	}
}

func (s *Server) Handler() http.Handler {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mux != nil {
		return corsMiddleware(s.mux)
	}
	return nil
}

func jsonResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, statusCode int, message string) {
	jsonResponse(w, statusCode, map[string]string{"error": message})
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	status := s.clientMgr.GetAuthStatus(r.Context())
	jsonResponse(w, http.StatusOK, status)
}

func (s *Server) handleHasCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	status := s.clientMgr.GetAuthStatus(r.Context())
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"is_configured":   status.Authorized && status.Authenticated,
		"has_credentials": status.HasCredentials,
		"language":        i18n.CurrentLanguage(),
	})
}

func (s *Server) handleSaveCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		ApiID   string `json:"api_id"`
		ApiHash string `json:"api_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request JSON")
		return
	}

	if strings.TrimSpace(req.ApiID) == "" || strings.TrimSpace(req.ApiHash) == "" {
		jsonError(w, http.StatusBadRequest, "API ID and API Hash are required")
		return
	}

	if err := s.clientMgr.InitClient(req.ApiID, req.ApiHash); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	_ = s.storage.SaveConfigKey("api_id", req.ApiID)
	_ = s.storage.SaveConfigKey("api_hash", req.ApiHash)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Credentials saved successfully",
	})
}

func (s *Server) handleSendCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Phone       string `json:"phone"`
		PhoneNumber string `json:"phone_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request JSON")
		return
	}

	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		phone = strings.TrimSpace(req.PhoneNumber)
	}

	if phone == "" {
		jsonError(w, http.StatusBadRequest, "Phone number is required")
		return
	}

	if !s.clientMgr.IsInitialized() {
		cfg, _ := s.storage.LoadConfig(config.DefaultConfig())
		if cfg.ApiID != "" && cfg.ApiHash != "" {
			_ = s.clientMgr.InitClient(cfg.ApiID, cfg.ApiHash)
		}
	}

	phoneHash, err := s.clientMgr.SendCode(r.Context(), phone)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	_ = s.storage.SaveConfigKey("phone", phone)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"phone_code_hash": phoneHash,
	})
}

func (s *Server) handleSignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Phone         string `json:"phone"`
		PhoneNumber   string `json:"phone_number"`
		Code          string `json:"code"`
		PhoneCodeHash string `json:"phone_code_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request JSON")
		return
	}

	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		phone = strings.TrimSpace(req.PhoneNumber)
	}

	if !s.clientMgr.IsInitialized() {
		cfg, _ := s.storage.LoadConfig(config.DefaultConfig())
		if cfg.ApiID != "" && cfg.ApiHash != "" {
			_ = s.clientMgr.InitClient(cfg.ApiID, cfg.ApiHash)
		}
	}

	state, err := s.clientMgr.VerifyCode(r.Context(), phone, req.Code, req.PhoneCodeHash)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"state":   state,
	})
}

func (s *Server) handleTwoFactor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request JSON")
		return
	}

	if !s.clientMgr.IsInitialized() {
		cfg, _ := s.storage.LoadConfig(config.DefaultConfig())
		if cfg.ApiID != "" && cfg.ApiHash != "" {
			_ = s.clientMgr.InitClient(cfg.ApiID, cfg.ApiHash)
		}
	}

	if err := s.clientMgr.TwoFactor(r.Context(), req.Password); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"state":   "LOGGED_IN",
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	_ = s.clientMgr.InitClient("", "")
	_ = s.storage.SaveConfigKey("api_id", "")
	_ = s.storage.SaveConfigKey("api_hash", "")
	_ = s.storage.SaveConfigKey("phone", "")

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (s *Server) handleGetChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")

	chats, err := s.storage.GetChats(category, search)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	counts, _ := s.storage.GetChatCounts()

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"chats":  chats,
		"counts": counts,
	})
}

func (s *Server) handleSyncChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	logbus.Info(logbus.CatChats, i18n.T("chats.syncStarted"))
	chats, err := s.clientMgr.FetchDialogsAndSave(r.Context(), s.storage)
	if err != nil {
		logbus.Error(logbus.CatChats, fmt.Sprintf(i18n.T("telegram.fetchDialogsError"), err))
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logbus.Success(logbus.CatChats, fmt.Sprintf(i18n.T("chats.synced"), len(chats)))
	counts, _ := s.storage.GetChatCounts()

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"chats":   chats,
		"counts":  counts,
	})
}

func (s *Server) handleLeaveChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		ChatID  int64   `json:"chat_id"`
		ChatIDs []int64 `json:"chat_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request JSON")
		return
	}

	if len(req.ChatIDs) > 0 {
		results := s.clientMgr.LeaveOrDeleteChatsBatch(r.Context(), s.storage, req.ChatIDs)
		successCount := 0
		for _, err := range results {
			if err == nil {
				successCount++
			}
		}
		if successCount > 0 {
			logbus.Warn(logbus.CatChats, fmt.Sprintf(i18n.T("chats.batchRemoved"), successCount))
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"results": results,
		})
		return
	}

	if req.ChatID == 0 {
		jsonError(w, http.StatusBadRequest, "chat_id is required")
		return
	}

	chats, _ := s.storage.GetChats("everything", strconv.FormatInt(req.ChatID, 10))
	var chatTitle string
	for _, c := range chats {
		if c.ID == req.ChatID {
			chatTitle = c.Title
			break
		}
	}
	if chatTitle == "" {
		chatTitle = strconv.FormatInt(req.ChatID, 10)
	} else {
		chatTitle = fmt.Sprintf("%s (%d)", chatTitle, req.ChatID)
	}

	if err := s.clientMgr.LeaveOrDeleteChat(r.Context(), s.storage, req.ChatID); err != nil {
		logbus.Error(logbus.CatChats, fmt.Sprintf(i18n.T("telegram.leaveChannelError"), err), chatTitle)
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logbus.Warn(logbus.CatChats, fmt.Sprintf(i18n.T("chats.removed"), chatTitle))
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"chat_id": req.ChatID,
	})
}

func (s *Server) handleExportChats(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	if category == "" {
		category = "all"
	}

	content, err := s.storage.ExportChatsTXT(category)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fileName := fmt.Sprintf("telegrampv_%s.txt", category)

	// Guardar siempre el archivo en el directorio del programa
	dir, err := os.Getwd()
	if err != nil || dir == "" {
		if execPath, err := os.Executable(); err == nil {
			dir = filepath.Dir(execPath)
		} else {
			dir = "."
		}
	}

	filePath := filepath.Join(dir, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		log.Printf("[SERVER] Warning: Failed to write TXT export to %s: %v", filePath, err)
	} else {
		log.Printf("[SERVER] Saved TXT export to %s", filePath)
	}

	if r.Header.Get("Accept") == "application/json" || r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"filename": fileName,
			"path":     filePath,
		})
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func (s *Server) handleOpenTelegram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		jsonError(w, http.StatusBadRequest, "URL is required")
		return
	}

	if !strings.HasPrefix(req.URL, "tg://") && !strings.HasPrefix(req.URL, "https://t.me/") {
		jsonError(w, http.StatusBadRequest, "Invalid Telegram URL")
		return
	}

	err := openURL(req.URL)
	if err != nil {
		log.Printf("[SERVER] Error opening Telegram URL: %v", err)
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("%v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, _ := s.storage.LoadConfig(config.DefaultConfig())
		jsonResponse(w, http.StatusOK, cfg)
	case http.MethodPost:
		var cfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			jsonError(w, http.StatusBadRequest, "Invalid request JSON")
			return
		}

		before, _ := s.storage.LoadConfig(config.DefaultConfig())

		if cfg.Language != "" {
			i18n.SetLanguage(cfg.Language)
		}
		_ = s.storage.SaveConfig(cfg)
		logConfigChanges(before, cfg)

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	default:
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// Handlers del actualizador
func (s *Server) handleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		jsonError(w, http.StatusInternalServerError, "Updater service not available")
		return
	}

	rel, asset, err := s.updater.CheckForUpdate()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if rel != nil && asset != nil {
		s.mu.Lock()
		s.latestRel = rel
		s.mu.Unlock()
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"update_available": true,
			"latest":           rel.TagName,
			"version":          rel.TagName,
			"current":          config.AppVersion,
			"size_bytes":       asset.Size,
			"changelog":        rel.Body,
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"update_available": false,
		"latest":           nil,
		"current":          config.AppVersion,
		"size_bytes":       0,
	})
}

func (s *Server) handleUpdateProgress(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		jsonError(w, http.StatusInternalServerError, "Updater service not available")
		return
	}
	jsonResponse(w, http.StatusOK, s.updater.GetProgress())
}

func (s *Server) handleInstallUpdate(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		jsonError(w, http.StatusInternalServerError, "Updater service not available")
		return
	}

	s.mu.Lock()
	rel := s.latestRel
	s.mu.Unlock()

	var err error
	if rel == nil {
		rel, _, err = s.updater.CheckForUpdate()
		if err != nil || rel == nil {
			jsonError(w, http.StatusBadRequest, i18n.T("server.noUpdatesAvailable"))
			return
		}
	}

	if err := s.updater.InstallUpdate(rel); err != nil {
		logbus.Error(logbus.CatUpdater, i18n.T("updater.installStartError"), err.Error())
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	logbus.Info(logbus.CatUpdater, i18n.T("updater.installing", rel.TagName),
		i18n.T("updater.installingDetail", config.AppVersion))

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": i18n.T("server.startingInstall"),
	})
}

func (s *Server) handlePostponeUpdate(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		jsonError(w, http.StatusInternalServerError, "Updater service not available")
		return
	}

	var body struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Version == "" {
		jsonError(w, http.StatusUnprocessableEntity, i18n.T("server.versionRequired"))
		return
	}

	s.updater.Postpone(body.Version)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}
