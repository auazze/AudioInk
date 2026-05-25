package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// showModeChooser opens a small dialog letting the user pick between
// "Open in AudioInk" (returns 1) and "Auto-fix" (returns 2).
// Returns 0 if the window was closed without choosing.
func showModeChooser(fileCount int) int {
	app := NewApp()
	app.chooserMode = true
	app.chooserFileCount = fileCount

	err := wails.Run(&options.App{
		Title:         "AudioInk",
		Width:         340,
		Height:        260,
		MinWidth:      340,
		MinHeight:     260,
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
		logger.Printf("mode chooser error: %v", err)
	}

	return app.chooserChoice
}
