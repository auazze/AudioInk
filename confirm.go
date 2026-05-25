package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// showConfirmDialog hosts the manual-entry UI in a SEPARATE child process.
//
// Why: wails.Run is documented as one-per-process. In CLI auto-fix mode, the
// main process already used its wails.Run for the mode chooser. Calling a
// second wails.Run here (the original design) is undocumented and on
// Windows can hang due to WebView2 lifecycle quirks.
//
// Flow:
//  1. Write pending files to a temp JSON file.
//  2. exec ourselves with --confirm-only <pending> <results>.
//  3. Child runs runConfirmDialogChild → its OWN single wails.Run.
//  4. Child writes results to the second temp file and exits.
//  5. We read the results.
//
// If the child fails to launch or returns no usable output, every pending
// file ends up Skipped — the user can re-run the GUI to fix them later.
func showConfirmDialog(pending []PendingFile) []ManualEntry {
	results := make([]ManualEntry, len(pending))
	for i := range results {
		results[i] = ManualEntry{Skipped: true}
	}
	if len(pending) == 0 {
		return results
	}

	pendingFile, err := writeTempJSON("audioink-pending-*.json", pending)
	if err != nil {
		logger.Printf("confirm dialog: failed to write pending file: %v", err)
		return results
	}
	defer os.Remove(pendingFile)

	resultsFile := pendingFile + ".out"
	defer os.Remove(resultsFile)

	exe, err := os.Executable()
	if err != nil {
		logger.Printf("confirm dialog: cannot locate self: %v", err)
		return results
	}

	logger.Printf("spawning confirm dialog child: %s --confirm-only %s %s",
		filepath.Base(exe), filepath.Base(pendingFile), filepath.Base(resultsFile))

	cmd := exec.Command(exe, "--confirm-only", pendingFile, resultsFile)
	if err := cmd.Run(); err != nil {
		logger.Printf("confirm dialog child exited with error: %v", err)
		// Fall through — there may still be partial results.
	}

	data, err := os.ReadFile(resultsFile)
	if err != nil {
		logger.Printf("confirm dialog: no results file: %v", err)
		return results
	}
	var got []ManualEntry
	if err := json.Unmarshal(data, &got); err != nil {
		logger.Printf("confirm dialog: bad results JSON: %v", err)
		return results
	}
	if len(got) != len(pending) {
		logger.Printf("confirm dialog: results length mismatch (got %d, want %d)", len(got), len(pending))
		// Best effort: copy what we have, leave the rest as Skipped.
		for i, e := range got {
			if i < len(results) {
				results[i] = e
			}
		}
		return results
	}
	return got
}

// runConfirmDialogChild is the entry point invoked when the binary is
// re-executed with --confirm-only. It blocks on its own wails.Run and is
// the only place that wails.Run is called in the child process.
func runConfirmDialogChild(pendingPath, resultsPath string) int {
	initLogger()
	logger.Printf("=== confirm dialog child started: %s ===", filepath.Base(pendingPath))

	data, err := os.ReadFile(pendingPath)
	if err != nil {
		logger.Printf("child: read pending: %v", err)
		return 1
	}
	var pending []PendingFile
	if err := json.Unmarshal(data, &pending); err != nil {
		logger.Printf("child: parse pending: %v", err)
		return 1
	}
	if len(pending) == 0 {
		// Nothing to do — write empty array so parent doesn't complain.
		_ = os.WriteFile(resultsPath, []byte("[]"), 0644)
		return 0
	}

	app := NewApp()
	app.confirmMode = true
	app.pendingFiles = pending
	app.confirmResults = make([]ManualEntry, len(pending))
	for i := range app.confirmResults {
		app.confirmResults[i] = ManualEntry{Skipped: true}
	}

	err = wails.Run(&options.App{
		Title:         "AudioInk",
		Width:         460,
		Height:        440,
		MinWidth:      460,
		MinHeight:     440,
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
		logger.Printf("child: wails.Run error: %v", err)
		// Still attempt to write whatever results we have.
	}

	out, mErr := json.Marshal(app.confirmResults)
	if mErr != nil {
		logger.Printf("child: marshal results: %v", mErr)
		return 1
	}
	if err := os.WriteFile(resultsPath, out, 0644); err != nil {
		logger.Printf("child: write results: %v", err)
		return 1
	}
	logger.Printf("=== confirm dialog child done: %d results ===", len(app.confirmResults))
	return 0
}

// writeTempJSON marshals v and writes it to a fresh temp file matching
// pattern. Returns the absolute path.
func writeTempJSON(pattern string, v interface{}) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("create temp: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("encode: %w", err)
	}
	return f.Name(), nil
}
