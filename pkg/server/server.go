package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"telegrampv/pkg/config"
	"telegrampv/pkg/i18n"
	"telegrampv/pkg/logbus"
	"telegrampv/pkg/storage"
	"telegrampv/pkg/telegram"
)

type Server struct {
	clientMgr *telegram.ClientManager
	storage   *storage.Storage
	assets    fs.FS
	httpSrv   *http.Server
	mu        sync.Mutex
}

func NewServer(cm *telegram.ClientManager, st *storage.Storage, assets fs.FS) *Server {
	return &Server{
		clientMgr: cm,
		storage:   st,
		assets:    assets,
	}
}

func (s *Server) Start(host string, port int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpSrv != nil {
		_ = s.httpSrv.Close()
	}

	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/auth/status", s.handleAuthStatus)
	mux.HandleFunc("/api/auth/has-credentials", s.handleHasCredentials)
	mux.HandleFunc("/api/auth/credentials", s.handleSaveCredentials)
	mux.HandleFunc("/api/auth/send-code", s.handleSendCode)
	mux.HandleFunc("/api/auth/sign-in", s.handleSignIn)
	mux.HandleFunc("/api/auth/2fa", s.handleTwoFactor)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)

	mux.HandleFunc("/api/chats", s.handleGetChats)
	mux.HandleFunc("/api/chats/sync", s.handleSyncChats)
	mux.HandleFunc("/api/chats/leave", s.handleLeaveChat)
	mux.HandleFunc("/api/chats/export", s.handleExportChats)

	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/logs", s.handleLogs)

	// Static Files Handler
	if s.assets != nil {
		fileServer := http.FileServer(http.FS(s.assets))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}

			// Probar si el archivo existe en el FS embebido
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}

			_, err := s.assets.Open(path)
			if err != nil {
				// Fallback a index.html para SPA de Vue
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
		Handler: mux,
	}

	go func() {
		if err := s.httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[SERVER] Server error: %v", err)
		}
	}()

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
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request JSON")
		return
	}

	if strings.TrimSpace(req.Phone) == "" {
		jsonError(w, http.StatusBadRequest, "Phone number is required")
		return
	}

	phoneHash, err := s.clientMgr.SendCode(r.Context(), req.Phone)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = s.storage.SaveConfigKey("phone", req.Phone)

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
		Code          string `json:"code"`
		PhoneCodeHash string `json:"phone_code_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request JSON")
		return
	}

	state, err := s.clientMgr.VerifyCode(r.Context(), req.Phone, req.Code, req.PhoneCodeHash)
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

	chats, err := s.clientMgr.FetchDialogsAndSave(r.Context(), s.storage)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

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

	if err := s.clientMgr.LeaveOrDeleteChat(r.Context(), s.storage, req.ChatID); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"chat_id": req.ChatID,
	})
}

func (s *Server) handleExportChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

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
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
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

		if cfg.Language != "" {
			i18n.SetLanguage(cfg.Language)
		}
		_ = s.storage.SaveConfig(cfg)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	default:
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var sinceID int64
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		sinceID, _ = strconv.ParseInt(sinceStr, 10, 64)
	}

	entries, lastID := logbus.GetEntries(sinceID)
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"logs":    entries,
		"last_id": lastID,
	})
}
