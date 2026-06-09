package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// backupsDir is where originals are copied before a byte-destructive op so
// Undo can restore them. Lives next to history.json; honors AUDIOINK_DATA_DIR
// (tests) the same way historyPath() does.
func backupsDir() string {
	if override := os.Getenv("AUDIOINK_DATA_DIR"); override != "" {
		return filepath.Join(override, "backups")
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "AudioInk", "backups")
}

// backupForUndo copies src into the backups dir under a unique name and
// returns that path. Reuses copyFile (app.go).
func backupForUndo(src string) (string, error) {
	dir := backupsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create backups dir: %w", err)
	}
	dst := filepath.Join(dir, fmt.Sprintf("%s.%d.bak", filepath.Base(src), time.Now().UnixNano()))
	dst = uniquePath(dst)
	if err := copyFile(src, dst); err != nil {
		return "", fmt.Errorf("backup %s: %w", src, err)
	}
	return dst, nil
}

// restoreBackup reverts a destructive entry: delete the produced file, copy the
// backed-up original bytes back to OriginalPath, then remove the backup.
// Returns the restored path and whether it succeeded.
func restoreBackup(entry HistoryEntry) (string, bool) {
	if !pathsEqual(entry.NewPath, entry.OriginalPath) {
		if err := os.Remove(entry.NewPath); err != nil && !os.IsNotExist(err) {
			logger.Printf("  undo(destructive) remove produced file %s: %v", filepath.Base(entry.NewPath), err)
		}
	}
	if err := copyFile(entry.BackupPath, entry.OriginalPath); err != nil {
		logger.Printf("  undo(destructive) restore %s: %v", filepath.Base(entry.OriginalPath), err)
		return "", false
	}
	if err := os.Remove(entry.BackupPath); err != nil {
		logger.Printf("  undo(destructive) cleanup backup %s: %v", filepath.Base(entry.BackupPath), err)
	}
	logger.Printf("  undone(destructive): restored %s", filepath.Base(entry.OriginalPath))
	return entry.OriginalPath, true
}

// DestructiveRequest is one file for a byte-destructive op. OutName is the
// desired output filename (base only) — same as the source for trim/remux, or
// a new extension for convert.
type DestructiveRequest struct {
	FilePath string
	OutName  string
}

// destructiveOp performs the actual ffmpeg work writing src→dst.
type destructiveOp func(ctx context.Context, src, dst string, onProgress func(float64)) error

// processDestructive runs a byte-destructive op over the requests, mirroring
// processApply's Copy-vs-Overwrite semantics:
//   - Copy mode: output to the AudioInk/ subfolder, original untouched, no backup.
//   - Overwrite mode: back up the original, write to a temp file, then swap it
//     into place; record BackupPath so Undo can restore bytes.
//
// emit (may be nil) forwards per-file progress (0..1) to the UI via events.
func (a *App) processDestructive(reqs []DestructiveRequest, mode string, overwrite bool, op destructiveOp, emit func(filePath string, pct float64)) []ApplyResult {
	results := make([]ApplyResult, 0, len(reqs))
	var entries []HistoryEntry
	claimed := make(map[string]bool)

	var outputDir string
	if !overwrite && len(reqs) > 0 {
		outputDir = filepath.Join(filepath.Dir(reqs[0].FilePath), outputFolderName)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			logger.Printf("processDestructive: mkdir output: %v", err)
			for _, r := range reqs {
				results = append(results, ApplyResult{FilePath: r.FilePath, Error: fmt.Sprintf("create output dir: %v", err)})
			}
			return results
		}
	}

	ctx := context.Background()

	for _, req := range reqs {
		src := req.FilePath
		onProgress := func(p float64) {
			if emit != nil {
				emit(src, p)
			}
		}

		if _, err := os.Stat(src); os.IsNotExist(err) {
			results = append(results, ApplyResult{FilePath: src, Error: "file no longer exists"})
			continue
		}

		if overwrite {
			dir := filepath.Dir(src)
			finalPath := filepath.Join(dir, req.OutName)
			freed := map[string]bool{strings.ToLower(src): true}
			finalPath = uniquePathWithClaimed(finalPath, claimed, freed)

			// Keep the target extension on the temp file — ffmpeg picks the
			// output muxer from the extension, so a ".audioink-tmp" suffix
			// would make it fail with "Unable to find a suitable output format".
			ext := filepath.Ext(finalPath)
			tmp := uniquePath(strings.TrimSuffix(finalPath, ext) + ".audioink-tmp" + ext)
			if err := op(ctx, src, tmp, onProgress); err != nil {
				os.Remove(tmp)
				logger.Printf("  %s FAILED: %s: %v", mode, filepath.Base(src), err)
				results = append(results, ApplyResult{FilePath: src, Error: err.Error()})
				continue
			}

			backup, berr := backupForUndo(src)
			if berr != nil {
				os.Remove(tmp)
				results = append(results, ApplyResult{FilePath: src, Error: fmt.Sprintf("backup failed: %v", berr)})
				continue
			}
			if err := os.Remove(src); err != nil {
				os.Remove(tmp)
				os.Remove(backup)
				results = append(results, ApplyResult{FilePath: src, Error: fmt.Sprintf("replace original: %v", err)})
				continue
			}
			if err := os.Rename(tmp, finalPath); err != nil {
				// Best effort: original is gone but bytes are in the backup.
				os.Rename(backup, src)
				os.Remove(tmp)
				results = append(results, ApplyResult{FilePath: src, Error: fmt.Sprintf("finalize: %v", err)})
				continue
			}

			claimed[strings.ToLower(finalPath)] = true
			entries = append(entries, HistoryEntry{
				OriginalPath: src,
				NewPath:      finalPath,
				BackupPath:   backup,
			})
			results = append(results, ApplyResult{
				FilePath: src, NewPath: finalPath, NewFilename: filepath.Base(finalPath), Success: true,
			})
			logger.Printf("  %s(overwrite): %s", mode, filepath.Base(finalPath))
		} else {
			dst := uniquePathWithClaimed(filepath.Join(outputDir, req.OutName), claimed, nil)
			if err := op(ctx, src, dst, onProgress); err != nil {
				os.Remove(dst)
				logger.Printf("  %s FAILED: %s: %v", mode, filepath.Base(src), err)
				results = append(results, ApplyResult{FilePath: src, Error: err.Error()})
				continue
			}
			claimed[strings.ToLower(dst)] = true
			entries = append(entries, HistoryEntry{
				OriginalPath: src,
				NewPath:      dst,
				// No backup: original untouched in Copy mode. Undo just deletes
				// the produced copy.
				DeleteOnUndo: true,
			})
			results = append(results, ApplyResult{
				FilePath: src, NewPath: dst, NewFilename: filepath.Base(dst), Success: true,
			})
			logger.Printf("  %s(copy): %s", mode, filepath.Base(dst))
		}
	}

	recordBatch(mode, entries)
	a.registerMedia(collectResultPaths(results))
	return results
}

// collectResultPaths returns the produced paths of successful results so the
// player can preview freshly-created files.
func collectResultPaths(results []ApplyResult) []string {
	var paths []string
	for _, r := range results {
		if r.Success && r.NewPath != "" {
			paths = append(paths, r.NewPath)
		}
	}
	return paths
}
