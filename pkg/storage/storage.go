package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"telegrampv/pkg/config"
)

type Chat struct {
	ID           int64  `json:"id"`
	Type         string `json:"type"` // "group", "channel", "bot", "user"
	Title        string `json:"title"`
	Username     string `json:"username"`
	UnreadCount  int    `json:"unread_count"`
	MembersCount int    `json:"members_count"`
	AccessHash   int64  `json:"access_hash"`
	IsCreator    bool   `json:"is_creator"`
	IsAdmin      bool   `json:"is_admin"`
	UpdatedAt    int64  `json:"updated_at"`
}

type Storage struct {
	db *sql.DB
	mu sync.Mutex
}

func OpenResilient(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open sqlite database: %w", err)
	}

	st := &Storage{db: db}
	if err := st.initSchema(); err != nil {
		db.Close()
		// Intentar rescatar renombrando la base de datos corrupta
		backupPath := fmt.Sprintf("%s.corrupt_%d", dbPath, time.Now().Unix())
		_ = os.Rename(dbPath, backupPath)
		log.Printf("[STORAGE] Database corrupted, backed up to %s. Recreating clean DB...", backupPath)

		db2, err2 := sql.Open("sqlite", dbPath)
		if err2 != nil {
			return nil, fmt.Errorf("could not recreate sqlite database: %w", err2)
		}
		st.db = db2
		if err := st.initSchema(); err != nil {
			db2.Close()
			return nil, fmt.Errorf("could not init schema after recreation: %w", err)
		}
	}

	return st, nil
}

func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Storage) initSchema() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS chats (
			id INTEGER PRIMARY KEY,
			type TEXT NOT NULL,
			title TEXT NOT NULL,
			username TEXT DEFAULT '',
			unread_count INTEGER DEFAULT 0,
			members_count INTEGER DEFAULT 0,
			access_hash INTEGER DEFAULT 0,
			is_creator INTEGER DEFAULT 0,
			is_admin INTEGER DEFAULT 0,
			updated_at INTEGER DEFAULT 0
		);`,
		`CREATE INDEX IF NOT EXISTS idx_chats_type ON chats(type);`,
		`CREATE INDEX IF NOT EXISTS idx_chats_title ON chats(title);`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}

	// Migraciones defensivas para añadir columnas si la tabla de chats ya existía previamente
	_, _ = s.db.Exec("ALTER TABLE chats ADD COLUMN is_creator INTEGER DEFAULT 0;")
	_, _ = s.db.Exec("ALTER TABLE chats ADD COLUMN is_admin INTEGER DEFAULT 0;")

	return nil
}

func (s *Storage) GetConfigValue(key, defaultValue string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var val string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&val)
	if err != nil {
		return defaultValue
	}
	return val
}

func (s *Storage) SaveConfigKey(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("INSERT INTO config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}

func (s *Storage) LoadConfig(defaultCfg config.Config) (config.Config, error) {
	cfg := defaultCfg

	if lang := s.GetConfigValue("language", ""); lang != "" {
		cfg.Language = lang
	}
	if apiID := s.GetConfigValue("api_id", ""); apiID != "" {
		cfg.ApiID = apiID
	}
	if apiHash := s.GetConfigValue("api_hash", ""); apiHash != "" {
		cfg.ApiHash = apiHash
	}
	if phone := s.GetConfigValue("phone", ""); phone != "" {
		cfg.Phone = phone
	}
	if colorStr := s.GetConfigValue("color_id", ""); colorStr != "" {
		if cid, err := strconv.Atoi(colorStr); err == nil {
			cfg.ColorID = &cid
		}
	}
	if loaderColorStr := s.GetConfigValue("loader_color_id", ""); loaderColorStr != "" {
		if lcid, err := strconv.Atoi(loaderColorStr); err == nil {
			cfg.LoaderColorID = &lcid
		}
	}

	return cfg, nil
}

func (s *Storage) SaveConfig(cfg config.Config) error {
	if cfg.Language != "" {
		_ = s.SaveConfigKey("language", cfg.Language)
	}
	if cfg.ApiID != "" {
		_ = s.SaveConfigKey("api_id", cfg.ApiID)
	}
	if cfg.ApiHash != "" {
		_ = s.SaveConfigKey("api_hash", cfg.ApiHash)
	}
	if cfg.Phone != "" {
		_ = s.SaveConfigKey("phone", cfg.Phone)
	}
	if cfg.ColorID != nil {
		_ = s.SaveConfigKey("color_id", strconv.Itoa(*cfg.ColorID))
	}
	if cfg.LoaderColorID != nil {
		_ = s.SaveConfigKey("loader_color_id", strconv.Itoa(*cfg.LoaderColorID))
	}
	return nil
}

func (s *Storage) SaveChats(chats []Chat) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO chats (id, type, title, username, unread_count, members_count, access_hash, is_creator, is_admin, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			title = excluded.title,
			username = excluded.username,
			unread_count = excluded.unread_count,
			members_count = excluded.members_count,
			access_hash = excluded.access_hash,
			is_creator = excluded.is_creator,
			is_admin = excluded.is_admin,
			updated_at = excluded.updated_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, c := range chats {
		updatedAt := c.UpdatedAt
		if updatedAt == 0 {
			updatedAt = now
		}
		isCreatorInt := 0
		if c.IsCreator {
			isCreatorInt = 1
		}
		isAdminInt := 0
		if c.IsAdmin {
			isAdminInt = 1
		}
		_, err := stmt.Exec(c.ID, c.Type, c.Title, c.Username, c.UnreadCount, c.MembersCount, c.AccessHash, isCreatorInt, isAdminInt, updatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Storage) GetChats(category string, search string) ([]Chat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var query string
	var args []interface{}

	baseQuery := "SELECT id, type, title, username, unread_count, members_count, access_hash, is_creator, is_admin, updated_at FROM chats"
	var whereConditions []string

	category = strings.ToLower(strings.TrimSpace(category))
	switch category {
	case "all":
		whereConditions = append(whereConditions, "type IN ('group', 'channel')")
	case "groups":
		whereConditions = append(whereConditions, "type = 'group'")
	case "channels":
		whereConditions = append(whereConditions, "type = 'channel'")
	case "bots":
		whereConditions = append(whereConditions, "type = 'bot'")
	case "users":
		whereConditions = append(whereConditions, "type = 'user'")
	case "everything":
		// Todos los chats sin filtro de tipo
	default:
		whereConditions = append(whereConditions, "type IN ('group', 'channel')")
	}

	search = strings.TrimSpace(search)
	if search != "" {
		searchTerm := "%" + search + "%"
		whereConditions = append(whereConditions, "(title LIKE ? OR username LIKE ? OR CAST(id AS TEXT) LIKE ?)")
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	if len(whereConditions) > 0 {
		query = baseQuery + " WHERE " + strings.Join(whereConditions, " AND ")
	} else {
		query = baseQuery
	}

	query += " ORDER BY title COLLATE NOCASE ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var c Chat
		var isCreatorInt, isAdminInt int
		if err := rows.Scan(&c.ID, &c.Type, &c.Title, &c.Username, &c.UnreadCount, &c.MembersCount, &c.AccessHash, &isCreatorInt, &isAdminInt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.IsCreator = isCreatorInt != 0
		c.IsAdmin = isAdminInt != 0
		c.UnreadCount = 0
		c.MembersCount = 0
		if strings.TrimSpace(c.Title) == "" {
			if c.Username != "" {
				c.Title = c.Username
			} else {
				c.Title = "Deleted Account"
			}
		}
		chats = append(chats, c)
	}

	return chats, nil
}

func (s *Storage) DeleteChat(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM chats WHERE id = ?", id)
	return err
}

func (s *Storage) DeleteChatsBatch(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("DELETE FROM chats WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.Exec(id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Storage) ExportChatsTXT(category string) (string, error) {
	chats, err := s.GetChats(category, "")
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, c := range chats {
		sb.WriteString(fmt.Sprintf("%d - %s\n", c.ID, c.Title))
	}
	return sb.String(), nil
}

func (s *Storage) GetChatCounts() (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	counts := map[string]int{
		"all":      0,
		"groups":   0,
		"channels": 0,
		"bots":     0,
		"users":    0,
	}

	rows, err := s.db.Query("SELECT type, COUNT(*) FROM chats GROUP BY type")
	if err != nil {
		return counts, err
	}
	defer rows.Close()

	for rows.Next() {
		var t string
		var count int
		if err := rows.Scan(&t, &count); err == nil {
			switch t {
			case "group":
				counts["groups"] += count
				counts["all"] += count
			case "channel":
				counts["channels"] += count
				counts["all"] += count
			case "bot":
				counts["bots"] += count
			case "user":
				counts["users"] += count
			}
		}
	}

	return counts, nil
}
