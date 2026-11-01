package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"telegrampv/pkg/config"
	"telegrampv/pkg/i18n"
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

	runDesktopMode()
}

func hasArgument(target string) bool {
	for _, arg := range os.Args[1:] {
		if arg == target {
			return true
		}
	}
	return false
}

func printUsage() {
	fmt.Printf("TelegramPV v%s - Telegram Chat & Channel Manager\n", config.AppVersion)
	fmt.Println("Usage:")
	fmt.Println("  TelegramPV.exe            Run GUI Desktop mode")
	fmt.Println("  TelegramPV.exe --server   Run Headless HTTP Server mode")
	fmt.Println("  TelegramPV.exe --help     Show this help message")
}

func runServerMode() {
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

func runDesktopMode() {
	app, err := NewApp(assets)
	if err != nil {
		fmt.Printf("Error launching application: %v\n", err)
		os.Exit(1)
	}

	err = wails.Run(&options.App{
		Title:             "TelegramPV",
		Width:             1200,
		Height:            780,
		MinWidth:          900,
		MinHeight:         600,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 18, G: 20, B: 22, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
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
