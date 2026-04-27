package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	fingerbrowerApp := NewApp()

	err := wails.Run(&options.App{
		Title:     "fingerbrower",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
		},
		OnStartup: fingerbrowerApp.OnStartup,
		OnDomReady: fingerbrowerApp.OnDomReady,
		OnBeforeClose: fingerbrowerApp.OnBeforeClose,
		OnShutdown: fingerbrowerApp.OnShutdown,
	})
	if err != nil {
		log.Fatal(err)
	}
}
