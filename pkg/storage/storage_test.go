package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorageOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "telegrampv_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.sqlite3")
	st, err := OpenResilient(dbPath)
	if err != nil {
		t.Fatalf("failed to open storage: %v", err)
	}
	defer st.Close()

	// Test config
	err = st.SaveConfigKey("language", "es")
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	val := st.GetConfigValue("language", "en")
	if val != "es" {
		t.Fatalf("expected es, got %s", val)
	}

	// Test chat CRUD
	testChats := []Chat{
		{ID: -100123456, Type: "channel", Title: "Channel Test 1", AccessHash: 123},
		{ID: -987654, Type: "group", Title: "Group Test 1", AccessHash: 456},
		{ID: 111222, Type: "bot", Title: "Bot Test 1", AccessHash: 789},
	}

	if err := st.SaveChats(testChats); err != nil {
		t.Fatalf("failed to save chats: %v", err)
	}

	// Fetch groups and channels
	allChats, err := st.GetChats("all", "")
	if err != nil {
		t.Fatalf("failed to get all chats: %v", err)
	}
	if len(allChats) != 2 {
		t.Fatalf("expected 2 chats in 'all', got %d", len(allChats))
	}

	// Export TXT
	txt, err := st.ExportChatsTXT("all")
	if err != nil {
		t.Fatalf("failed to export chats txt: %v", err)
	}
	if txt == "" {
		t.Fatalf("exported txt should not be empty")
	}

	// Delete chat
	if err := st.DeleteChat(-987654); err != nil {
		t.Fatalf("failed to delete chat: %v", err)
	}

	groups, err := st.GetChats("groups", "")
	if err != nil {
		t.Fatalf("failed to get groups: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups after deletion, got %d", len(groups))
	}
}
