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

func historyPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "AudioInk", "history.json")
}

func loadHistory() History {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		return History{}
	}
	var h History
	json.Unmarshal(data, &h)
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

// undoLastBatch reverts the most recent batch, moves it to redo stack.
// Returns the list of original file paths for UI rescan.
func undoLastBatch() ([]string, error) {
	h := loadHistory()
	if len(h.Batches) == 0 {
		return nil, fmt.Errorf("no history to undo")
	}

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

	// Move to redo stack
	h.Batches = h.Batches[:len(h.Batches)-1]
	h.RedoStack = append(h.RedoStack, batch)
	if err := saveHistory(h); err != nil {
		logger.Printf("failed to save history after undo: %v", err)
	}

	return paths, nil
}

// redoLastBatch re-applies the most recently undone batch.
// Returns the list of new file paths for UI rescan.
func redoLastBatch() ([]string, error) {
	h := loadHistory()
	if len(h.RedoStack) == 0 {
		return nil, fmt.Errorf("no redo available")
	}

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

	// Move back to batches
	h.RedoStack = h.RedoStack[:len(h.RedoStack)-1]
	h.Batches = append(h.Batches, batch)
	if err := saveHistory(h); err != nil {
		logger.Printf("failed to save history after redo: %v", err)
	}

	return paths, nil
}
