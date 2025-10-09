package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:web/dist
var embeddedAssets embed.FS

func main() {
	webFS, err := fs.Sub(embeddedAssets, "web/dist")
	if err != nil {
		log.Fatal(err)
	}

	app := NewApp()

	if err := wails.Run(&options.App{
		Title:  "Arcaluminis",
		Width:  1200,
		Height: 800,
		// ✅ Embed built frontend for production builds:
		AssetServer: &assetserver.Options{Assets: webFS},
		Debug:       options.Debug{OpenInspectorOnStartup: true}, // 👈 open devtools in build
		OnStartup:   app.startup,
		OnShutdown:  app.shutdown,
		Bind:        []interface{}{app},
	}); err != nil {
		log.Fatal(err)
	}
}
