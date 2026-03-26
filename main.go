package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// CLI mode: audioink --fix file1.mp3 file2.flac ...
	if len(os.Args) > 1 && os.Args[1] == "--fix" {
		paths := collectFixPaths(os.Args[2:])
		if paths == nil {
			return // follower or error
		}

		choice := showModeChooser(len(paths))
		switch choice {
		case 1: // Open in AudioInk GUI
			launchGUI(paths)
		case 2: // Auto-fix
			os.Exit(fixPaths(paths, true))
		}
		return
	}

	launchGUI(nil)
}

func launchGUI(initialPaths []string) {
	app := NewApp()
	app.initialPaths = initialPaths

	err := wails.Run(&options.App{
		Title:         "AudioInk",
		Width:         1100,
		Height:        650,
		MinWidth:      600,
		MinHeight:     400,
		DisableResize: false,
		Frameless:     false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 15, B: 20, A: 1},
		OnStartup:        app.startup,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
			CSSDropProperty:    "--wails-drop-target",
			CSSDropValue:       "drop",
		},
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
			Theme:                windows.Dark,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
