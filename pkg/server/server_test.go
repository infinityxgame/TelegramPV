package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"telegrampv/pkg/storage"
	"telegrampv/pkg/telegram"
)

func TestHandleSaveCredentials(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.sqlite3")
	st, err := storage.OpenResilient(dbPath)
	if err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer st.Close()

	cm := telegram.NewClientManager()
	srv := NewServer(cm, st, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/credentials", srv.handleSaveCredentials)
	handler := corsMiddleware(mux)

	// Test OPTIONS preflight
	reqOpt := httptest.NewRequest(http.MethodOptions, "/api/auth/credentials", nil)
	recOpt := httptest.NewRecorder()
	handler.ServeHTTP(recOpt, reqOpt)

	if recOpt.Code != http.StatusNoContent {
		t.Errorf("Expected 204 for OPTIONS, got %d", recOpt.Code)
	}

	// Test POST credentials
	body := map[string]string{
		"api_id":   "123456",
		"api_hash": "0123456789abcdef0123456789abcdef",
	}
	jsonBody, _ := json.Marshal(body)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/auth/credentials", bytes.NewReader(jsonBody))
	reqPost.Header.Set("Content-Type", "application/json")
	recPost := httptest.NewRecorder()
	handler.ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusOK {
		t.Errorf("Expected 200 for POST, got %d: %s", recPost.Code, recPost.Body.String())
	}

	var res map[string]interface{}
	if err := json.NewDecoder(recPost.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if res["success"] != true {
		t.Errorf("Expected success=true, got %v", res["success"])
	}

	_ = os.Remove(dbPath)
}
