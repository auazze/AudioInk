//go:build linux

package main

import (
	"os/exec"
	"strings"
)

func promptManualEntry(filename string) ManualEntry {
	// Try zenity first (GNOME/GTK)
	if path, err := exec.LookPath("zenity"); err == nil {
		return promptZenity(path, filename)
	}
	// Try kdialog (KDE)
	if path, err := exec.LookPath("kdialog"); err == nil {
		return promptKdialog(path, filename)
	}
	logger.Println("    no dialog tool found (zenity/kdialog), skipping manual entry")
	return ManualEntry{Skipped: true}
}

func promptZenity(zenity, filename string) ManualEntry {
	cmd := exec.Command(zenity, "--forms",
		"--title=AudioInk — Manual Entry",
		"--text="+filename,
		"--add-entry=Artist",
		"--add-entry=Title",
		"--separator=|",
	)
	out, err := cmd.Output()
	if err != nil {
		// Exit code 1 = cancel
		return ManualEntry{Skipped: true}
	}
	return parseDialogOutput(string(out))
}

func promptKdialog(kdialog, filename string) ManualEntry {
	// kdialog doesn't have multi-field forms, so two sequential dialogs
	artistCmd := exec.Command(kdialog, "--inputbox", filename+"\n\nArtist:", "", "--title", "AudioInk — Manual Entry")
	artistOut, err := artistCmd.Output()
	if err != nil {
		return ManualEntry{Skipped: true}
	}
	artist := strings.TrimSpace(string(artistOut))

	titleCmd := exec.Command(kdialog, "--inputbox", filename+"\n\nTitle:", "", "--title", "AudioInk — Manual Entry")
	titleOut, err := titleCmd.Output()
	if err != nil {
		return ManualEntry{Skipped: true}
	}
	title := strings.TrimSpace(string(titleOut))

	if artist == "" && title == "" {
		return ManualEntry{Skipped: true}
	}
	return ManualEntry{Artist: artist, Title: title}
}
