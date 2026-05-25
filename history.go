package main

import (
	"AudioInk/tagger"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const maxHistoryBatches = 50

type HistoryEntry struct {
	OriginalPath string      `json:"originalPath"`
	NewPath      string      `json:"newPath"`
	OriginalTags tagger.Tags `json:"originalTags"`
	NewTags      tagger.Tags `json:"newTags"`
}

type HistoryBatch struct {
	Timestamp string         `json:"timestamp"`
	Mode      string         `json:"mode"`
	Count     int            `json:"count"`
	Entries   []HistoryEntry `json:"entries"`
}

type History struct {
	Batches   []HistoryBatch `json:"batches"`
	RedoStack []HistoryBatch `json:"redoStack,omitempty"`
}

// historyPath returns the on-disk location of the undo/redo history.
//
// Tests MUST redirect this via the AUDIOINK_DATA_DIR env var (typically
// to t.TempDir()) — otherwise `go test ./...` writes batches into the
// user's real %APPDATA%/AudioInk/history.json, polluting their actual
// undo history with stale entries that point to deleted temp dirs.
func historyPath() string {
	if override := os.Getenv("AUDIOINK_DATA_DIR"); override != "" {
		return filepath.Join(override, "history.json")
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "AudioInk", "history.json")
}

func loadHistory() History {
	p := historyPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return History{}
	}
	var h History
	if err := json.Unmarshal(data, &h); err != nil {
		// Corrupt history (interrupted write, manual edit, etc.). Back it
		// up before treating as empty so the user has a chance to recover
		// undo state instead of silently losing everything.
		backup := p + ".corrupt." + time.Now().Format("20060102-150405")
		if writeErr := os.WriteFile(backup, data, 0644); writeErr == nil {
			if logger != nil {
				logger.Printf("history corrupt (%v); backed up to %s", err, filepath.Base(backup))
			}
		} else if logger != nil {
			logger.Printf("history corrupt (%v); backup also failed: %v", err, writeErr)
		}
		return History{}
	}
	return h
}

func saveHistory(h History) error {
	if len(h.Batches) > maxHistoryBatches {
		h.Batches = h.Batches[len(h.Batches)-maxHistoryBatches:]
	}

	p := historyPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}
	return os.WriteFile(p, data, 0644)
}

// recordBatch saves a completed batch and clears the redo stack.
func recordBatch(mode string, entries []HistoryEntry) {
	if len(entries) == 0 {
		return
	}
	h := loadHistory()
	h.RedoStack = nil // new operation invalidates redo
	h.Batches = append(h.Batches, HistoryBatch{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Mode:      mode,
		Count:     len(entries),
		Entries:   entries,
	})
	if err := saveHistory(h); err != nil {
		logger.Printf("failed to save history: %v", err)
	} else {
		logger.Printf("history saved: %d entries (%s)", len(entries), mode)
	}
}

// undoLastBatch reverts the most recent batch whose files still exist.
//
// Stale batches (where every entry's NewPath has been deleted/moved/
// renamed by a subsequent operation) are SILENTLY POPPED and discarded —
// they're useless to the user because we can't recover anything. We keep
// popping until we find a batch with at least one reverable file, or we
// run out of history.
//
// Returns the list of original file paths for UI rescan, and an error
// only when there is genuinely nothing left to undo.
func undoLastBatch() ([]string, error) {
	h := loadHistory()
	staleDiscarded := 0

	for len(h.Batches) > 0 {
		batch := h.Batches[len(h.Batches)-1]
		var paths []string

		for i := len(batch.Entries) - 1; i >= 0; i-- {
			entry := batch.Entries[i]
			currentPath := entry.NewPath

			if _, err := os.Stat(currentPath); os.IsNotExist(err) {
				logger.Printf("  undo skip (missing): %s", filepath.Base(currentPath))
				continue
			}

			if err := tagger.Write(currentPath, entry.OriginalTags); err != nil {
				logger.Printf("  undo tag error %s: %v", filepath.Base(currentPath), err)
				continue
			}

			finalPath := currentPath
			if currentPath != entry.OriginalPath {
				if err := os.Rename(currentPath, entry.OriginalPath); err != nil {
					logger.Printf("  undo rename error %s: %v", filepath.Base(currentPath), err)
				} else {
					finalPath = entry.OriginalPath
				}
			}

			paths = append(paths, finalPath)
			logger.Printf("  undone: %s → %s", filepath.Base(currentPath), filepath.Base(finalPath))
		}

		// Pop this batch off — either it was successful (move to redo)
		// or stale (drop entirely so it doesn't keep wasting Undo clicks).
		h.Batches = h.Batches[:len(h.Batches)-1]
		if len(paths) > 0 {
			h.RedoStack = append(h.RedoStack, batch)
			if err := saveHistory(h); err != nil {
				logger.Printf("failed to save history after undo: %v", err)
			}
			if staleDiscarded > 0 {
				logger.Printf("undo: discarded %d stale batch(es) before this one", staleDiscarded)
			}
			return paths, nil
		}
		staleDiscarded++
		logger.Printf("  batch was stale (no recoverable files) — discarding and trying next")
	}

	if err := saveHistory(h); err != nil {
		logger.Printf("failed to save cleaned history: %v", err)
	}
	if staleDiscarded > 0 {
		return nil, fmt.Errorf("nothing to undo — discarded %d stale batch(es); history is now clean", staleDiscarded)
	}
	return nil, fmt.Errorf("no history to undo")
}

// redoLastBatch re-applies the most recently undone batch whose source
// files still exist. Stale entries (where OriginalPath has been
// deleted/moved since the undo) are popped off the redo stack until we
// find one that's actually applicable.
func redoLastBatch() ([]string, error) {
	h := loadHistory()
	staleDiscarded := 0

	for len(h.RedoStack) > 0 {
		batch := h.RedoStack[len(h.RedoStack)-1]
		var paths []string

		for _, entry := range batch.Entries {
			srcPath := entry.OriginalPath

			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				logger.Printf("  redo skip (missing): %s", filepath.Base(srcPath))
				continue
			}

			if err := tagger.Write(srcPath, entry.NewTags); err != nil {
				logger.Printf("  redo tag error %s: %v", filepath.Base(srcPath), err)
				continue
			}

			finalPath := srcPath
			if entry.NewPath != entry.OriginalPath {
				if err := os.Rename(srcPath, entry.NewPath); err != nil {
					logger.Printf("  redo rename error %s: %v", filepath.Base(srcPath), err)
				} else {
					finalPath = entry.NewPath
				}
			}

			paths = append(paths, finalPath)
			logger.Printf("  redone: %s → %s", filepath.Base(srcPath), filepath.Base(finalPath))
		}

		h.RedoStack = h.RedoStack[:len(h.RedoStack)-1]
		if len(paths) > 0 {
			h.Batches = append(h.Batches, batch)
			if err := saveHistory(h); err != nil {
				logger.Printf("failed to save history after redo: %v", err)
			}
			if staleDiscarded > 0 {
				logger.Printf("redo: discarded %d stale batch(es) before this one", staleDiscarded)
			}
			return paths, nil
		}
		staleDiscarded++
		logger.Printf("  batch was stale (no applicable files) — discarding and trying next")
	}

	if err := saveHistory(h); err != nil {
		logger.Printf("failed to save cleaned history: %v", err)
	}
	if staleDiscarded > 0 {
		return nil, fmt.Errorf("nothing to redo — discarded %d stale batch(es); redo stack is now clean", staleDiscarded)
	}
	return nil, fmt.Errorf("no redo available")
}
