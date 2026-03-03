package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"AudioInk/parser"
	"AudioInk/scanner"
	"AudioInk/tagger"
)

// runFix processes audio files in headless CLI mode.
// It writes corrected metadata tags and renames files to match the parsed structure.
// Returns 0 on full success, 1 if any errors occurred or no files were provided.
func runFix(paths []string) int {
	supported := scanner.FilterSupported(paths)

	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "audioink: no files provided")
		return 1
	}
	if len(supported) == 0 {
		fmt.Fprintln(os.Stderr, "audioink: no supported audio files found")
		return 1
	}

	successes := 0
	errors := 0

	for _, f := range supported {
		if err := fixOneFile(f); err != nil {
			fmt.Fprintf(os.Stderr, "audioink: %s: %v\n", filepath.Base(f), err)
			errors++
		} else {
			successes++
		}
	}

	total := len(supported)
	showNotification(successes, errors, total)

	if errors > 0 {
		return 1
	}
	return 0
}

// fixOneFile parses the filename, writes corrected tags, and renames the file.
func fixOneFile(filePath string) error {
	pr := parser.Parse(filePath)

	// Build artist: main artist + featured joined with " & "
	allArtists := []string{}
	if pr.Artist != "" {
		allArtists = append(allArtists, pr.Artist)
	}
	allArtists = append(allArtists, pr.FeaturedArtists...)
	artist := strings.Join(allArtists, " & ")

	// Build tag title: title + extras in parens if any
	tagTitle := pr.Title
	if pr.Extras != "" {
		tagTitle = tagTitle + " (" + pr.Extras + ")"
	}

	// Write tags
	tags := tagger.Tags{
		Artist: artist,
		Title:  tagTitle,
		Track:  pr.Track,
	}
	if err := tagger.Write(filePath, tags); err != nil {
		return fmt.Errorf("write tags: %w", err)
	}

	// Build new filename and rename if different
	ext := filepath.Ext(pr.Filename)
	newFilename := buildNewFilename(artist, pr.Title, pr.Extras, ext)
	if newFilename == "" {
		return nil
	}

	newPath := filepath.Join(filepath.Dir(filePath), newFilename)
	if newPath != filePath {
		newPath = uniquePath(newPath)
		if err := os.Rename(filePath, newPath); err != nil {
			return fmt.Errorf("rename: %w", err)
		}
	}

	return nil
}
