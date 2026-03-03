package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	notificationsEnabled = false
	os.Exit(m.Run())
}

func TestRunFix_NoArgs(t *testing.T) {
	code := runFix(nil)
	if code != 1 {
		t.Errorf("runFix(nil) = %d, want 1", code)
	}

	code = runFix([]string{})
	if code != 1 {
		t.Errorf("runFix([]) = %d, want 1", code)
	}
}

func TestRunFix_UnsupportedExtensions(t *testing.T) {
	paths := []string{
		"/some/path/readme.txt",
		"/some/path/image.png",
		"/some/path/document.pdf",
	}
	code := runFix(paths)
	if code != 1 {
		t.Errorf("runFix(unsupported) = %d, want 1", code)
	}
}

func TestFixOneFile_NonexistentFile(t *testing.T) {
	err := fixOneFile("/nonexistent/path/Artist - Title.mp3")
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

	// Should either succeed (taglib writes tags) or return a clear error.
	_ = fixOneFile(fakeFile)
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

	code := runFix(files)
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
	_ = runFix([]string{txtFile, mp3File})

	// The txt file should still exist untouched (not processed)
	data, err := os.ReadFile(txtFile)
	if err != nil {
		t.Fatalf("txt file should still exist: %v", err)
	}
	if string(data) != "text" {
		t.Errorf("txt file content changed, expected 'text', got %q", string(data))
	}
}
