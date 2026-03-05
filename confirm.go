package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func showConfirmDialog(pending []PendingFile) []ManualEntry {
	app := NewApp()
	app.confirmMode = true
	app.pendingFiles = pending
	app.confirmResults = make([]ManualEntry, len(pending))
	app.confirmDone = make(chan struct{})
	for i := range app.confirmResults {
		app.confirmResults[i] = ManualEntry{Skipped: true}
	}

	err := wails.Run(&options.App{
		Title:         "AudioInk",
		Width:         450,
		Height:        340,
		MinWidth:      450,
		MinHeight:     340,
		DisableResize: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 15, B: 20, A: 1},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			Theme:                windows.Dark,
		},
	})
	if err != nil {
		logger.Printf("confirm dialog error: %v", err)
	}

	return app.confirmResults
}
