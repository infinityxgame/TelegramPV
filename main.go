package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"telegrampv/pkg/config"
	"telegrampv/pkg/i18n"
	"telegrampv/pkg/updater"
)

//go:embed all:dashboard/dist
var assets embed.FS

func main() {
	i18n.SetLanguage(i18n.DetectSystemLanguage())

	if hasArgument("--help") || hasArgument("-h") {
		printUsage()
		return
	}

	if hasArgument("--server") {
		runServerMode()
		return
	}

	if hasArgument("--update") {
		os.Exit(runUpdateMode())
	}

	runDesktopMode()
}

func hasArgument(target string) bool {
	for _, arg := range os.Args[1:] {
		if strings.EqualFold(arg, target) {
			return true
		}
	}
	return false
}

func printUsage() {
	fmt.Printf("TelegramPV v%s - Telegram Chat & Channel Manager\n", config.AppVersion)
	fmt.Println(i18n.T("cli.usageHeader"))
	fmt.Println(i18n.T("cli.usageDesktop"))
	fmt.Println(i18n.T("cli.usageServer"))
	fmt.Println(i18n.T("cli.usageUpdate"))
}

func checkConflictAndExit() bool {
	if isTelegramDLRunning() {
		showNotice("TelegramDL se esta ejecutando. No es posible abrir TelegramPV al mismo tiempo porque comparten la misma sesion de Telegram. Por favor, cierra TelegramDL antes de abrir TelegramPV.")
		return true
	}
	return false
}

func runServerMode() {
	if checkConflictAndExit() {
		os.Exit(1)
	}

	fmt.Println("Starting TelegramPV in Headless Server Mode...")
	app, err := NewApp(assets)
	if err != nil {
		fmt.Printf("Fatal error initializing app: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("TelegramPV Web Server running at: %s\n", app.GetServerURL())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down TelegramPV Server...")
	app.shutdown(context.Background())
}

func runUpdateMode() int {
	appUpdater := updater.NewAppUpdater()
	release, asset, err := appUpdater.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("cli.checkError"), err)
		return 1
	}
	if release == nil || asset == nil {
		fmt.Printf(i18n.T("cli.alreadyUpdated"), config.AppVersion)
		return 0
	}

	fmt.Printf(i18n.T("cli.updating"), config.AppVersion, release.TagName, asset.Name)
	if err := appUpdater.InstallUpdate(release); err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("cli.installError"), err)
		return 1
	}

	lastStatus := ""
	for {
		progress := appUpdater.GetProgress()
		if progress.Status != lastStatus {
			if strings.HasPrefix(progress.Status, "error:") {
				fmt.Fprintln(os.Stderr, progress.Status)
				return 1
			}
			if progress.Status != "" {
				fmt.Printf(i18n.T("cli.progressState"), progress.Status, progress.Percentage)
			}
			lastStatus = progress.Status
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func runDesktopMode() {
	if checkConflictAndExit() {
		return
	}

	app, err := NewApp(assets)
	if err != nil {
		fmt.Printf("Error launching application: %v\n", err)
		os.Exit(1)
	}

	err = wails.Run(&options.App{
		Title:             "TelegramPV",
		Width:             1200,
		Height:            640,
		MinWidth:          900,
		MinHeight:         600,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 18, G: 20, B: 22, A: 255},
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: app.Handler(),
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		fmt.Printf("Wails runtime error: %v\n", err)
	}
}
