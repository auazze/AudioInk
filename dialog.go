package main

import "strings"

// ManualEntry holds user-provided artist/title from a manual entry dialog.
type ManualEntry struct {
	Artist  string
	Title   string
	Skipped bool
}

// parseDialogOutput parses the stdout from a native dialog.
// Expected format: "artist|title" or "SKIP".
func parseDialogOutput(output string) ManualEntry {
	output = strings.TrimSpace(output)
	if output == "" || strings.EqualFold(output, "SKIP") {
		return ManualEntry{Skipped: true}
	}
	parts := strings.SplitN(output, "|", 2)
	if len(parts) != 2 {
		return ManualEntry{Skipped: true}
	}
	artist := strings.TrimSpace(parts[0])
	title := strings.TrimSpace(parts[1])
	if artist == "" && title == "" {
		return ManualEntry{Skipped: true}
	}
	return ManualEntry{Artist: artist, Title: title}
}
