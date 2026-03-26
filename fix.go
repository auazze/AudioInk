package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// collectFixPaths aggregates file paths from multiple context-menu processes.
// The first process becomes the "leader" and collects all paths; followers
// append their paths and exit.  Returns nil for followers.
func collectFixPaths(args []string) []string {
	initLogger()
	logger.Printf("=== --fix called with %d paths ===", len(args))

	if len(args) == 0 {
		logger.Println("no files provided")
		return nil
	}

	queuePath := filepath.Join(os.TempDir(), "audioink-fix-queue.txt")
	lockPath := filepath.Join(os.TempDir(), "audioink-fix.lock")

	// Clean stale lock (older than 30s — previous crash)
	if info, err := os.Stat(lockPath); err == nil {
		if time.Since(info.ModTime()) > 30*time.Second {
			os.Remove(lockPath)
		}
	}

	appendToQueue(queuePath, args)

	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		logger.Println("follower: paths queued, exiting")
		return nil
	}
	lock.Close()
	defer os.Remove(lockPath)

	logger.Println("leader: waiting for queue to stabilize")
	var lastSize int64 = -1
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		info, err := os.Stat(queuePath)
		if err != nil {
			break
		}
		if info.Size() == lastSize {
			break
		}
		lastSize = info.Size()
	}

	allPaths := readQueue(queuePath)
	os.Remove(queuePath)

	if len(allPaths) == 0 {
		logger.Println("no paths in queue")
		return nil
	}

	logger.Printf("leader: collected %d files", len(allPaths))
	return allPaths
}

func appendToQueue(queuePath string, paths []string) {
	f, err := os.OpenFile(queuePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	for _, p := range paths {
		f.WriteString(p + "\n")
	}
}

func readQueue(queuePath string) []string {
	data, err := os.ReadFile(queuePath)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	var result []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !seen[line] {
			seen[line] = true
			result = append(result, line)
		}
	}
	return result
}

// fixPaths parses, optionally auto-fixes, and applies tags to all files.
// When autoFix is true, files with both metadata artist+title are applied
// directly (artist as-is, title title-cased); only files without metadata
// go to the confirm dialog.
func fixPaths(paths []string, autoFix bool) int {
	initLogger()
	supported := scanner.FilterSupported(paths)

	if len(supported) == 0 {
		logger.Printf("no supported audio files (got %d unsupported)", len(paths))
		return 1
	}

	logger.Printf("%d supported files, autoFix=%v", len(supported), autoFix)

	parsed := make([]parsedFile, 0, len(supported))
	var pending []PendingFile
	var pendingIndices []int

	for _, f := range supported {
		pr := parser.Parse(f)

		// Read and clean metadata
		var metaArtist, metaTitle string
		if tags, err := tagger.Read(f); err == nil {
			logger.Printf("  tags for %s: artist=%q title=%q", filepath.Base(f), tags.Artist, tags.Title)
			cleanA, cleanT, metaFeat, _ := parser.CleanMetadata(tags.Artist, tags.Title)
			metaAll := []string{}
			if cleanA != "" {
				metaAll = append(metaAll, cleanA)
			}
			metaAll = append(metaAll, metaFeat...)
			metaArtist = strings.Join(metaAll, " & ")
			metaTitle = cleanT
		} else {
			logger.Printf("  read tags error for %s: %v", filepath.Base(f), err)
		}

		// Filename-parsed artist
		allArtists := []string{}
		if pr.Artist != "" {
			allArtists = append(allArtists, pr.Artist)
		}
		allArtists = append(allArtists, pr.FeaturedArtists...)
		fileArtist := strings.Join(allArtists, " & ")

		// Prefer metadata casing when content matches
		useArtist := fileArtist
		useTitle := pr.Title
		if metaArtist != "" && strings.EqualFold(useArtist, metaArtist) {
			useArtist = metaArtist
		}
		if metaTitle != "" && strings.EqualFold(useTitle, metaTitle) {
			useTitle = metaTitle
		}

		needsManual := pr.Confidence == parser.Low

		if autoFix && metaArtist != "" && metaTitle != "" {
			useArtist = metaArtist // as-is from metadata
			useTitle = metaTitle   // as-is; TitleCase already applied by parser if from filename
			// Only title-case if the metadata title is ALL CAPS or all lowercase
			if metaTitle == strings.ToLower(metaTitle) || metaTitle == strings.ToUpper(metaTitle) {
				useTitle = parser.TitleCase(metaTitle)
			}
			needsManual = false
			logger.Printf("  auto-fix: %s → %q - %q", filepath.Base(f), useArtist, useTitle)
		}

		pf := parsedFile{
			filePath: f,
			artist:   useArtist,
			title:    useTitle,
			extras:   pr.Extras,
			track:    pr.Track,
			filename: pr.Filename,
			lowConf:  needsManual,
		}
		parsed = append(parsed, pf)

		if needsManual {
			displayArtist := fileArtist
			displayTitle := pr.Title
			if displayArtist == "" && metaArtist != "" {
				displayArtist = metaArtist
			}
			if displayTitle == "" && metaTitle != "" {
				displayTitle = metaTitle
			}

			pendingIndices = append(pendingIndices, len(parsed)-1)
			pending = append(pending, PendingFile{
				Filename:   pr.Filename,
				Artist:     displayArtist,
				Title:      displayTitle,
				MetaArtist: metaArtist,
				MetaTitle:  metaTitle,
				FileArtist: fileArtist,
				FileTitle:  pr.Title,
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
	var historyEntries []HistoryEntry

	for _, pf := range parsed {
		if pf.lowConf {
			logger.Printf("  skipped: %s", filepath.Base(pf.filePath))
			continue
		}
		entry, err := fixOneFile(pf)
		if err != nil {
			logger.Printf("  ERROR %s: %v", filepath.Base(pf.filePath), err)
			errors++
		} else {
			historyEntries = append(historyEntries, entry)
			successes++
		}
	}

	mode := "cli-fix"
	if autoFix {
		mode = "autofix"
	}
	recordBatch(mode, historyEntries)

	total := len(supported)
	showNotification(successes, errors, total)
	logger.Printf("=== Done: %d/%d successful ===", successes, total)

	if errors > 0 {
		return 1
	}
	return 0
}

func fixOneFile(pf parsedFile) (HistoryEntry, error) {
	logger.Printf("  fixing: %s", filepath.Base(pf.filePath))

	// Read original tags for undo history
	origTags, _ := tagger.Read(pf.filePath)

	tagTitle := pf.title
	if pf.extras != "" {
		tagTitle = tagTitle + " (" + pf.extras + ")"
	}

	logger.Printf("    artist=%q title=%q track=%d", pf.artist, tagTitle, pf.track)

	newTags := tagger.Tags{
		Artist: pf.artist,
		Title:  tagTitle,
		Track:  pf.track,
	}
	if err := tagger.Write(pf.filePath, newTags); err != nil {
		return HistoryEntry{}, fmt.Errorf("write tags: %w", err)
	}

	ext := filepath.Ext(pf.filename)
	newFilename := buildNewFilename(pf.artist, pf.title, pf.extras, ext)
	finalPath := pf.filePath

	if newFilename != "" {
		newPath := filepath.Join(filepath.Dir(pf.filePath), newFilename)
		if newPath == pf.filePath {
			logger.Println("    name already correct, tags updated")
		} else if pathsEqual(newPath, pf.filePath) {
			logger.Printf("    rename (case fix) -> %s", filepath.Base(newPath))
			if err := os.Rename(pf.filePath, newPath); err != nil {
				return HistoryEntry{}, fmt.Errorf("rename: %w", err)
			}
			finalPath = newPath
		} else {
			newPath = uniquePath(newPath)
			logger.Printf("    rename -> %s", filepath.Base(newPath))
			if err := os.Rename(pf.filePath, newPath); err != nil {
				return HistoryEntry{}, fmt.Errorf("rename: %w", err)
			}
			finalPath = newPath
		}
	}

	return HistoryEntry{
		OriginalPath: pf.filePath,
		NewPath:      finalPath,
		OriginalTags: origTags,
		NewTags:      newTags,
	}, nil
}
