package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// fakeOp writes fixed content to dst, simulating an ffmpeg transcode without
// needing the binary.
func fakeOp(content string) destructiveOp {
	return func(ctx context.Context, src, dst string, onProgress func(float64)) error {
		if onProgress != nil {
			onProgress(1)
		}
		return os.WriteFile(dst, []byte(content), 0644)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestDestructiveOverwriteThenUndo(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	src := filepath.Join(dir, "song.mp3")
	writeFile(t, src, "ORIGINAL-BYTES")

	reqs := []DestructiveRequest{{FilePath: src, OutName: "song.mp3"}}
	results := app.processDestructive(reqs, "convert", true, fakeOp("CONVERTED-BYTES"), nil)

	if len(results) != 1 || !results[0].Success {
		t.Fatalf("expected 1 success, got %+v", results)
	}
	if got := readFile(t, src); got != "CONVERTED-BYTES" {
		t.Fatalf("after convert, file = %q, want CONVERTED-BYTES", got)
	}

	// The history batch must carry a backup path.
	h := loadHistory()
	if len(h.Batches) == 0 {
		t.Fatal("no history batch recorded")
	}
	entry := h.Batches[len(h.Batches)-1].Entries[0]
	if entry.BackupPath == "" {
		t.Fatal("overwrite destructive op must record a BackupPath")
	}
	if _, err := os.Stat(entry.BackupPath); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	// Undo restores the original bytes and removes the backup.
	paths, err := undoLastBatch()
	if err != nil {
		t.Fatalf("undo error: %v", err)
	}
	if len(paths) != 1 || paths[0] != src {
		t.Fatalf("undo paths = %v, want [%s]", paths, src)
	}
	if got := readFile(t, src); got != "ORIGINAL-BYTES" {
		t.Fatalf("after undo, file = %q, want ORIGINAL-BYTES", got)
	}
	if _, err := os.Stat(entry.BackupPath); !os.IsNotExist(err) {
		t.Errorf("backup should be removed after undo")
	}
}

func TestDestructiveOverwriteChangedExtensionUndo(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	src := filepath.Join(dir, "song.wav")
	writeFile(t, src, "WAV-ORIGINAL")

	// Convert .wav → .mp3 (filename changes).
	reqs := []DestructiveRequest{{FilePath: src, OutName: "song.mp3"}}
	results := app.processDestructive(reqs, "convert", true, fakeOp("MP3-DATA"), nil)
	if !results[0].Success {
		t.Fatalf("convert failed: %+v", results[0])
	}
	newPath := filepath.Join(dir, "song.mp3")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("converted file missing: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("original .wav should be gone after overwrite convert")
	}

	if _, err := undoLastBatch(); err != nil {
		t.Fatalf("undo error: %v", err)
	}
	// Original .wav restored, produced .mp3 removed.
	if got := readFile(t, src); got != "WAV-ORIGINAL" {
		t.Fatalf("after undo, .wav = %q, want WAV-ORIGINAL", got)
	}
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Errorf("produced .mp3 should be removed on undo")
	}
}

func TestDestructiveCopyThenUndo(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	src := filepath.Join(dir, "song.mp3")
	writeFile(t, src, "ORIGINAL")

	reqs := []DestructiveRequest{{FilePath: src, OutName: "song.flac"}}
	results := app.processDestructive(reqs, "convert", false, fakeOp("FLAC-COPY"), nil)
	if !results[0].Success {
		t.Fatalf("copy convert failed: %+v", results[0])
	}

	// Original untouched.
	if got := readFile(t, src); got != "ORIGINAL" {
		t.Fatalf("copy mode must not touch original, got %q", got)
	}
	copyPath := results[0].NewPath
	if filepath.Dir(copyPath) != filepath.Join(dir, outputFolderName) {
		t.Fatalf("copy should land in AudioInk/ subfolder, got %s", copyPath)
	}
	if _, err := os.Stat(copyPath); err != nil {
		t.Fatalf("copy missing: %v", err)
	}

	// Undo deletes the produced copy; original stays.
	if _, err := undoLastBatch(); err != nil {
		t.Fatalf("undo error: %v", err)
	}
	if _, err := os.Stat(copyPath); !os.IsNotExist(err) {
		t.Errorf("copy should be deleted on undo")
	}
	if got := readFile(t, src); got != "ORIGINAL" {
		t.Errorf("original must remain after copy-undo, got %q", got)
	}
}

func TestCleanTagsRoutesThroughBackup(t *testing.T) {
	// Verify CleanID3's plumbing records an undoable batch even though the
	// op here is a no-op copy (taglib not exercised — that's an E2E concern).
	app := NewApp()
	dir := t.TempDir()
	src := filepath.Join(dir, "x.mp3")
	writeFile(t, src, "DATA")

	op := func(ctx context.Context, s, d string, onP func(float64)) error {
		return copyFile(s, d)
	}
	results := app.processDestructive(
		[]DestructiveRequest{{FilePath: src, OutName: "x.mp3"}}, "clean", true, op, nil)
	if !results[0].Success {
		t.Fatalf("clean failed: %+v", results[0])
	}
	if _, err := undoLastBatch(); err != nil {
		t.Fatalf("undo error: %v", err)
	}
	if got := readFile(t, src); got != "DATA" {
		t.Errorf("after undo, file = %q, want DATA", got)
	}
}
