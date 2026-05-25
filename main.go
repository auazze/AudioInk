package main

import (
	"embed"
	"encoding/json"
	"os"
	"os/exec"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Internal: re-exec entry for the confirm dialog (CLI auto-fix path).
	// Keeps wails.Run one-per-process. See confirm.go.
	if len(os.Args) == 4 && os.Args[1] == "--confirm-only" {
		os.Exit(runConfirmDialogChild(os.Args[2], os.Args[3]))
	}

	// Internal: re-exec entry for the GUI launched FROM the chooser.
	// The chooser process owns one wails.Run (the chooser window itself).
	// To then open the GUI we spawn a fresh process whose wails.Run is
	// the GUI window — never two within the same process.
	if len(os.Args) == 3 && os.Args[1] == "--gui-files" {
		paths := readPathsFile(os.Args[2])
		_ = os.Remove(os.Args[2])
		launchGUI(paths)
		return
	}

	// CLI mode: audioink --fix file1.mp3 file2.flac ...
	if len(os.Args) > 1 && os.Args[1] == "--fix" {
		paths := collectFixPaths(os.Args[2:])
		if paths == nil {
			return // follower or error
		}

		choice := showModeChooser(len(paths))
		switch choice {
		case 1: // Open in AudioInk GUI — spawn fresh process, see comment above.
			spawnGUIChild(paths)
		case 2: // Auto-fix in this same process; confirm dialog (if needed) is
			// itself a child process, so we still stay at one wails.Run here.
			os.Exit(fixPaths(paths, true))
		}
		return
	}

	launchGUI(nil)
}

// spawnGUIChild writes paths to a temp file and re-execs ourselves with
// --gui-files <tempfile>. Returns once the child has been started; the
// parent (chooser) process exits naturally afterwards.
func spawnGUIChild(paths []string) {
	exe, err := os.Executable()
	if err != nil {
		logger.Printf("spawnGUIChild: locate self: %v", err)
		return
	}

	f, err := os.CreateTemp("", "audioink-gui-paths-*.json")
	if err != nil {
		logger.Printf("spawnGUIChild: temp file: %v", err)
		return
	}
	enc := json.NewEncoder(f)
	if err := enc.Encode(paths); err != nil {
		f.Close()
		os.Remove(f.Name())
		logger.Printf("spawnGUIChild: encode paths: %v", err)
		return
	}
	f.Close()

	cmd := exec.Command(exe, "--gui-files", f.Name())
	if err := cmd.Start(); err != nil {
		logger.Printf("spawnGUIChild: start: %v", err)
		os.Remove(f.Name())
		return
	}
	// Don't wait — the child cleans up the temp file itself.
}

func readPathsFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
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
