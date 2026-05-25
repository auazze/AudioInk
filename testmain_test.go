package main

import (
	"os"
	"testing"
)

// TestMain redirects history.json and audioink.log to a per-run temp
// directory so `go test ./...` does NOT pollute the user's real
// %APPDATA%/AudioInk/ with batches that point to deleted t.TempDir()
// paths. Without this, every test run leaves stale undo batches in
// production that the user's Undo button has to discard.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "audioink-test-data-*")
	if err != nil {
		// Fall back to no isolation rather than aborting — the test
		// will still run, just with the side effects we're trying to
		// avoid.
		os.Exit(m.Run())
	}
	os.Setenv("AUDIOINK_DATA_DIR", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
