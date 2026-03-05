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

type parsedFile struct {
	filePath string
	artist   string
	title    string
	extras   string
	track    int
	filename string
	lowConf  bool
}

func runFix(paths []string) int {
	initLogger()
	logger.Printf("=== --fix called with %d paths ===", len(paths))

	supported := scanner.FilterSupported(paths)

	if len(paths) == 0 {
		logger.Println("no files provided")
		return 1
	}
	if len(supported) == 0 {
		logger.Printf("no supported audio files (got %d unsupported)", len(paths))
		return 1
	}

	logger.Printf("%d supported files", len(supported))

	parsed := make([]parsedFile, 0, len(supported))
	var pending []PendingFile
	var pendingIndices []int

	for _, f := range supported {
		pr := parser.Parse(f)

		allArtists := []string{}
		if pr.Artist != "" {
			allArtists = append(allArtists, pr.Artist)
		}
		allArtists = append(allArtists, pr.FeaturedArtists...)
		artist := strings.Join(allArtists, " & ")

		pf := parsedFile{
			filePath: f,
			artist:   artist,
			title:    pr.Title,
			extras:   pr.Extras,
			track:    pr.Track,
			filename: pr.Filename,
			lowConf:  pr.Confidence == parser.Low,
		}
		parsed = append(parsed, pf)

		if pf.lowConf {
			pendingIndices = append(pendingIndices, len(parsed)-1)
			pending = append(pending, PendingFile{
				Filename: pr.Filename,
				Artist:   artist,
				Title:    pr.Title,
			})
		}
	}

	if len(pending) > 0 {
		results := showConfirmDialog(pending)
		for i, idx := range pendingIndices {
			if i < len(results) {
				if results[i].Skipped {
					parsed[idx].lowConf = true
				} else {
					parsed[idx].artist = results[i].Artist
					parsed[idx].title = results[i].Title
					parsed[idx].lowConf = false
				}
			}
		}
	}

	successes := 0
	errors := 0

	for _, pf := range parsed {
		if pf.lowConf {
			logger.Printf("  skipped (manual entry): %s", filepath.Base(pf.filePath))
			continue
		}
		if err := fixOneFile(pf); err != nil {
			logger.Printf("  ERROR %s: %v", filepath.Base(pf.filePath), err)
			errors++
		} else {
			successes++
		}
	}

	total := len(supported)
	showNotification(successes, errors, total)
	logger.Printf("=== Done: %d/%d successful ===", successes, total)

	if errors > 0 {
		return 1
	}
	return 0
}

func fixOneFile(pf parsedFile) error {
	logger.Printf("  fixing: %s", filepath.Base(pf.filePath))

	tagTitle := pf.title
	if pf.extras != "" {
		tagTitle = tagTitle + " (" + pf.extras + ")"
	}

	logger.Printf("    parsed: artist=%q title=%q track=%d",
		pf.artist, tagTitle, pf.track)

	tags := tagger.Tags{
		Artist: pf.artist,
		Title:  tagTitle,
		Track:  pf.track,
	}
	if err := tagger.Write(pf.filePath, tags); err != nil {
		return fmt.Errorf("write tags: %w", err)
	}

	ext := filepath.Ext(pf.filename)
	newFilename := buildNewFilename(pf.artist, pf.title, pf.extras, ext)
	if newFilename == "" {
		logger.Println("    no rename needed (empty filename)")
		return nil
	}

	newPath := filepath.Join(filepath.Dir(pf.filePath), newFilename)
	if newPath == pf.filePath {
		logger.Println("    name already correct, tags updated")
	} else if pathsEqual(newPath, pf.filePath) {
		logger.Printf("    rename (case fix) -> %s", filepath.Base(newPath))
		if err := os.Rename(pf.filePath, newPath); err != nil {
			return fmt.Errorf("rename: %w", err)
		}
	} else {
		newPath = uniquePath(newPath)
		logger.Printf("    rename -> %s", filepath.Base(newPath))
		if err := os.Rename(pf.filePath, newPath); err != nil {
			return fmt.Errorf("rename: %w", err)
		}
	}

	return nil
}
