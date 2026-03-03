//go:build darwin

package main

import (
	"os/exec"
	"strings"
)

func promptManualEntry(filename string) ManualEntry {
	escaped := strings.ReplaceAll(filename, `"`, `\"`)

	artistScript := `display dialog "` + escaped + `" & return & return & "Artist:" default answer "" buttons {"Skip", "OK"} default button "OK" with title "AudioInk — Manual Entry"
if button returned of result is "Skip" then
	return "SKIP"
else
	return text returned of result
end if`

	artistCmd := exec.Command("osascript", "-e", artistScript)
	artistOut, err := artistCmd.Output()
	if err != nil {
		return ManualEntry{Skipped: true}
	}
	artist := strings.TrimSpace(string(artistOut))
	if artist == "SKIP" {
		return ManualEntry{Skipped: true}
	}

	titleScript := `display dialog "` + escaped + `" & return & return & "Title:" default answer "" buttons {"Skip", "OK"} default button "OK" with title "AudioInk — Manual Entry"
if button returned of result is "Skip" then
	return "SKIP"
else
	return text returned of result
end if`

	titleCmd := exec.Command("osascript", "-e", titleScript)
	titleOut, err := titleCmd.Output()
	if err != nil {
		return ManualEntry{Skipped: true}
	}
	title := strings.TrimSpace(string(titleOut))
	if title == "SKIP" {
		return ManualEntry{Skipped: true}
	}

	if artist == "" && title == "" {
		return ManualEntry{Skipped: true}
	}
	return ManualEntry{Artist: artist, Title: title}
}
