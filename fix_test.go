package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFix_NoArgs(t *testing.T) {
	code := fixPaths(nil, false)
	if code != 1 {
		t.Errorf("fixPaths(nil, false) = %d, want 1", code)
	}

	code = fixPaths([]string{}, false)
	if code != 1 {
		t.Errorf("fixPaths([]) = %d, want 1", code)
	}
}

func TestRunFix_UnsupportedExtensions(t *testing.T) {
	paths := []string{
		"/some/path/readme.txt",
		"/some/path/image.png",
		"/some/path/document.pdf",
	}
	code := fixPaths(paths, false)
	if code != 1 {
		t.Errorf("fixPaths(unsupported) = %d, want 1", code)
	}
}

func TestFixOneFile_NonexistentFile(t *testing.T) {
	pf := parsedFile{
		filePath: "/nonexistent/path/Artist - Title.mp3",
		artist:   "Artist",
		title:    "Title",
		filename: "Artist - Title.mp3",
	}
	_, err := fixOneFile(pf)
	if err == nil {
		t.Fatal("fixOneFile on nonexistent file should return an error")
	}
	if !strings.Contains(err.Error(), "write tags") {
		t.Errorf("error should mention 'write tags', got: %v", err)
	}
}

func TestFixOneFile_EmptyFileNoRename(t *testing.T) {
	// An empty .mp3 file may or may not fail at taglib depending on format.
	// At minimum, verify fixOneFile doesn't panic and handles the file.
	dir := t.TempDir()
	fakeFile := filepath.Join(dir, "Artist - Title.mp3")
	if err := os.WriteFile(fakeFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create fake file: %v", err)
	}

	pf := parsedFile{
		filePath: fakeFile,
		artist:   "Artist",
		title:    "Title",
		filename: "Artist - Title.mp3",
	}
	_, _ = fixOneFile(pf)
}

func TestRunFix_AllFilesFail(t *testing.T) {
	// Create temp files that look like supported audio but are empty (no valid headers).
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "Daft Punk - Get Lucky.mp3"),
		filepath.Join(dir, "Queen - Bohemian Rhapsody.flac"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("not real audio"), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", f, err)
		}
	}

	code := fixPaths(files, false)
	if code != 1 {
		t.Errorf("runFix (all files fail) = %d, want 1", code)
	}
}

func TestRunFix_MixedSupportedUnsupported(t *testing.T) {
	// Provide a mix: unsupported files are filtered out, supported are processed.
	// Taglib may succeed on .mp3 with arbitrary content, so we just verify
	// that unsupported files are properly filtered (not processed).
	dir := t.TempDir()
	mp3File := filepath.Join(dir, "Artist - Title.mp3")
	txtFile := filepath.Join(dir, "readme.txt")
	if err := os.WriteFile(mp3File, []byte("not real"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(txtFile, []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should not crash; txt file should be filtered out
	_ = fixPaths([]string{txtFile, mp3File}, false)

	// The txt file should still exist untouched (not processed)
	data, err := os.ReadFile(txtFile)
	if err != nil {
		t.Fatalf("txt file should still exist: %v", err)
	}
	if string(data) != "text" {
		t.Errorf("txt file content changed, expected 'text', got %q", string(data))
	}
}
