package main

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"telegrampv/pkg/config"
	"telegrampv/pkg/i18n"
	"telegrampv/pkg/logbus"
	"telegrampv/pkg/server"
	"telegrampv/pkg/storage"
	"telegrampv/pkg/telegram"
)

type App struct {
	ctx       context.Context
	clientMgr *telegram.ClientManager
	storage   *storage.Storage
	server    *server.Server
	config    config.Config
	assets    fs.FS
	serverURL string
	mu        sync.Mutex
}

func NewApp(assets fs.FS) (*App, error) {
	config.InitPaths()

	logFileErr := logbus.Init(config.DataDir)
	logbus.Attach()
	logbus.Success(logbus.CatSystem, i18n.T("system.appStarted", config.AppVersion))
	if logFileErr != nil {
		logbus.Warn(logbus.CatSystem, i18n.T("system.logFileError"), logFileErr.Error())
	}

	dbPath := filepath.Join(config.DataDir, "telegrampv.sqlite3")
	st, err := storage.OpenResilient(dbPath)
	if err != nil {
		logbus.Error(logbus.CatSystem, i18n.T("system.dbOpenFatal"), err.Error())
		return nil, err
	}

	config.SecureDataDir(dbPath, dbPath+"-wal", dbPath+"-shm")

	cfg, err := st.LoadConfig(config.DefaultConfig())
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if cfg.Language != "" {
		i18n.SetLanguage(cfg.Language)
	}

	cm := telegram.NewClientManager()

	if cfg.ApiID != "" && cfg.ApiHash != "" {
		_ = cm.InitClient(cfg.ApiID, cfg.ApiHash)
	}

	srv := server.NewServer(cm, st, assets)
	serverURL, err := srv.Start(config.GetServerHost(), config.GetServerPort())
	if err != nil {
		logbus.Warn(logbus.CatSystem, fmt.Sprintf("Server port error: %v, retrying with auto port", err))
		serverURL, _ = srv.Start(config.GetServerHost(), 0)
	}

	app := &App{
		clientMgr: cm,
		storage:   st,
		server:    srv,
		config:    cfg,
		assets:    assets,
		serverURL: serverURL,
	}

	return app, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	wailsRuntime.WindowSetTitle(ctx, fmt.Sprintf("TelegramPV v%s", config.AppVersion))
}

func (a *App) shutdown(ctx context.Context) {
	if a.server != nil {
		a.server.Stop()
	}
	if a.clientMgr != nil {
		a.clientMgr.Stop()
	}
	if a.storage != nil {
		_ = a.storage.Close()
	}
}

func (a *App) GetServerURL() string {
	return a.serverURL
}
