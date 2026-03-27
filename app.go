package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"AudioInk/parser"
	"AudioInk/scanner"
	"AudioInk/tagger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const outputFolderName = "AudioInk"

var logger *log.Logger

func initLogger() {
	if logger != nil {
		return // already initialized
	}
	exe, _ := os.Executable()
	logPath := filepath.Join(filepath.Dir(exe), "audioink.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		f, _ = os.OpenFile("audioink.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	}
	logger = log.New(f, "", log.Ldate|log.Ltime)
}

type PendingFile struct {
	Filename   string `json:"filename"`
	Artist     string `json:"artist"`
	Title      string `json:"title"`
	MetaArtist string `json:"metaArtist"`
	MetaTitle  string `json:"metaTitle"`
	FileArtist string `json:"fileArtist"`
	FileTitle  string `json:"fileTitle"`
}

type App struct {
	ctx              context.Context
	confirmMode      bool
	chooserMode      bool
	chooserFileCount int
	chooserChoice    int // 0=none, 1=gui, 2=autofix
	pendingFiles     []PendingFile
	confirmResults   []ManualEntry
	confirmDone      chan struct{}
	initialPaths     []string
}

func NewApp() *App {
	initLogger()
	return &App{}
}

func (a *App) IsConfirmMode() bool {
	return a.confirmMode
}

func (a *App) GetPendingFiles() []PendingFile {
	return a.pendingFiles
}

func (a *App) ConfirmSubmit(index int, artist, title string) {
	if index >= 0 && index < len(a.confirmResults) {
		a.confirmResults[index] = ManualEntry{Artist: artist, Title: title}
	}
}

func (a *App) ConfirmSkip(index int) {
	if index >= 0 && index < len(a.confirmResults) {
		a.confirmResults[index] = ManualEntry{Skipped: true}
	}
}

func (a *App) ConfirmDone() {
	if a.confirmDone != nil {
		close(a.confirmDone)
	}
	runtime.Quit(a.ctx)
}

// ConfirmBatchFromTags applies metadata to all pending files from fromIndex onward.
// useArtist/useTitle control which fields are taken from metadata (vs. parsed filename).
func (a *App) ConfirmBatchFromTags(fromIndex int, useArtist, useTitle bool) {
	for i := fromIndex; i < len(a.pendingFiles); i++ {
		pf := a.pendingFiles[i]
		artist := pf.Artist
		title := pf.Title

		if useArtist && pf.MetaArtist != "" {
			artist = pf.MetaArtist
		}
		if useTitle && pf.MetaTitle != "" {
			title = pf.MetaTitle
		}

		if artist == "" && title == "" {
			a.confirmResults[i] = ManualEntry{Skipped: true}
		} else {
			a.confirmResults[i] = ManualEntry{Artist: artist, Title: title}
		}
	}
}

// --- Mode chooser methods (context menu → choose GUI or auto-fix) ---

func (a *App) IsChooserMode() bool       { return a.chooserMode }
func (a *App) GetChooserFileCount() int   { return a.chooserFileCount }
func (a *App) ChooseGUI()                 { a.chooserChoice = 1; runtime.Quit(a.ctx) }
func (a *App) ChooseAutoFix()             { a.chooserChoice = 2; runtime.Quit(a.ctx) }

// --- Initial files (context menu → "Open in GUI") ---

func (a *App) GetInitialFiles() []FileResult {
	if len(a.initialPaths) == 0 {
		return nil
	}
	logger.Printf("loading %d initial files from context menu", len(a.initialPaths))
	return a.processFiles(a.initialPaths)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.Println("app startup complete")
}

type FileResult struct {
	FilePath        string   `json:"filePath"`
	Filename        string   `json:"filename"`
	Artist          string   `json:"artist"`
	Title           string   `json:"title"`
	Track           int      `json:"track"`
	FeaturedArtists []string `json:"featuredArtists"`
	Extras          string   `json:"extras"`
	Confidence      string   `json:"confidence"`
	CurrentArtist   string   `json:"currentArtist"`
	CurrentTitle    string   `json:"currentTitle"`
	NewFilename     string   `json:"newFilename"`
	CleanMetaArtist string   `json:"cleanMetaArtist"`
	CleanMetaTitle  string   `json:"cleanMetaTitle"`
}

func buildNewFilename(artist, title, extras, ext string) string {
	name := ""
	if artist != "" && title != "" {
		name = artist + " - " + title
	} else if title != "" {
		name = title
	}
	if name == "" {
		return ""
	}
	if extras != "" {
		name = name + " (" + extras + ")"
	}
	return sanitizeFilename(name) + ext
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"<", "", ">", "", ":", "", "\"", "",
		"/", "", "\\", "", "|", "", "?", "", "*", "",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	for strings.Contains(name, "  ") {
		name = strings.ReplaceAll(name, "  ", " ")
	}
	return name
}

func toFileResult(pr parser.ParseResult) FileResult {
	existing, err := tagger.Read(pr.FilePath)

	var cleanMetaArtist, cleanMetaTitle string
	if err != nil {
		logger.Printf("  read tags error for %s: %v", pr.Filename, err)
	} else {
		logger.Printf("  existing tags: artist=%q title=%q", existing.Artist, existing.Title)

		// Clean metadata for UI display / buttons (no auto-apply — user decides)
		cleanA, cleanT, metaFeat, _ := parser.CleanMetadata(existing.Artist, existing.Title)
		metaAll := []string{}
		if cleanA != "" {
			metaAll = append(metaAll, cleanA)
		}
		metaAll = append(metaAll, metaFeat...)
		cleanMetaArtist = strings.Join(metaAll, " & ")
		cleanMetaTitle = cleanT
	}

	allArtists := []string{}
	if pr.Artist != "" {
		allArtists = append(allArtists, pr.Artist)
	}
	allArtists = append(allArtists, pr.FeaturedArtists...)
	artist := strings.Join(allArtists, " & ")

	title := pr.Title
	extras := pr.Extras

	// Prefer metadata casing when content matches (metadata is more authoritative)
	if cleanMetaArtist != "" && strings.EqualFold(artist, cleanMetaArtist) {
		artist = cleanMetaArtist
	}
	if cleanMetaTitle != "" && strings.EqualFold(title, cleanMetaTitle) {
		title = cleanMetaTitle
	}

	ext := filepath.Ext(pr.Filename)
	newFilename := buildNewFilename(artist, title, extras, ext)

	logger.Printf("  parsed: artist=%q title=%q track=%d confidence=%s -> %s",
		artist, title, pr.Track, pr.Confidence, newFilename)

	return FileResult{
		FilePath:        pr.FilePath,
		Filename:        pr.Filename,
		Artist:          artist,
		Title:           title,
		Track:           pr.Track,
		FeaturedArtists: pr.FeaturedArtists,
		Extras:          pr.Extras,
		Confidence:      pr.Confidence.String(),
		CurrentArtist:   existing.Artist,
		CurrentTitle:    existing.Title,
		NewFilename:     newFilename,
		CleanMetaArtist: cleanMetaArtist,
		CleanMetaTitle:  cleanMetaTitle,
	}
}

func (a *App) SelectFiles() ([]FileResult, error) {
	logger.Println("SelectFiles called")
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select audio files",
		Filters: []runtime.FileFilter{
			{DisplayName: "Audio Files", Pattern: "*.mp3;*.flac;*.ogg;*.m4a;*.wav;*.wma;*.opus"},
		},
	})
	if err != nil {
		logger.Printf("file dialog error: %v", err)
		return nil, err
	}
	if len(paths) == 0 {
		logger.Println("file dialog cancelled")
		return nil, nil
	}
	logger.Printf("selected %d files", len(paths))
	return a.processFiles(paths), nil
}

func (a *App) SelectDirectory() ([]FileResult, error) {
	logger.Println("SelectDirectory called")
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select folder with audio files",
	})
	if err != nil {
		logger.Printf("directory dialog error: %v", err)
		return nil, err
	}
	if dir == "" {
		logger.Println("directory dialog cancelled")
		return nil, nil
	}
	logger.Printf("selected directory: %s", dir)
	return a.ScanDirectory(dir)
}

func (a *App) processFiles(paths []string) []FileResult {
	supported := scanner.FilterSupported(paths)
	logger.Printf("  %d supported files", len(supported))
	results := make([]FileResult, 0, len(supported))
	for _, f := range supported {
		logger.Printf("parsing: %s", filepath.Base(f))
		pr := parser.Parse(f)
		results = append(results, toFileResult(pr))
	}
	return results
}

func (a *App) ScanDirectory(dir string) ([]FileResult, error) {
	logger.Printf("scanning directory: %s", dir)
	files, err := scanner.ScanDirectory(dir)
	if err != nil {
		logger.Printf("scan error: %v", err)
		return nil, fmt.Errorf("scan directory: %w", err)
	}
	logger.Printf("found %d audio files", len(files))

	results := make([]FileResult, 0, len(files))
	for _, f := range files {
		logger.Printf("parsing: %s", filepath.Base(f))
		pr := parser.Parse(f)
		results = append(results, toFileResult(pr))
	}
	return results, nil
}

func (a *App) ScanFiles(paths []string) []FileResult {
	logger.Printf("ScanFiles called with %d paths", len(paths))
	for _, p := range paths {
		logger.Printf("  input path: %s", p)
	}
	return a.processFiles(paths)
}

type ApplyRequest struct {
	FilePath string `json:"filePath"`
	Artist   string `json:"artist"`
	Title    string `json:"title"`
	Extras   string `json:"extras"`
	Track    int    `json:"track"`
}

type ApplyResult struct {
	FilePath    string `json:"filePath"`
	NewPath     string `json:"newPath"`
	NewFilename string `json:"newFilename"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// ApplyTagsCopy copies files to AudioInk/ subfolder with new names + tags
func (a *App) ApplyTagsCopy(requests []ApplyRequest) []ApplyResult {
	logger.Printf("=== ApplyTagsCopy called with %d files ===", len(requests))

	if len(requests) == 0 {
		logger.Println("no files to process")
		return nil
	}

	sourceDir := filepath.Dir(requests[0].FilePath)
	outputDir := filepath.Join(sourceDir, outputFolderName)
	logger.Printf("output directory: %s", outputDir)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Printf("failed to create output dir: %v", err)
		errMsg := fmt.Sprintf("failed to create output directory: %v", err)
		results := make([]ApplyResult, len(requests))
		for i, req := range requests {
			results[i] = ApplyResult{FilePath: req.FilePath, Error: errMsg}
		}
		return results
	}

	return a.processApply(requests, outputDir, false)
}

// ApplyTagsOverwrite renames + writes tags on originals in place
func (a *App) ApplyTagsOverwrite(requests []ApplyRequest) []ApplyResult {
	logger.Printf("=== ApplyTagsOverwrite called with %d files ===", len(requests))

	if len(requests) == 0 {
		logger.Println("no files to process")
		return nil
	}

	return a.processApply(requests, "", true)
}

// ApplyQuick applies tags and renames a single file in place.
// Used by the frontend ManualEntryDialog for garbage filenames.
func (a *App) ApplyQuick(filePath, artist, title, extras string) ApplyResult {
	logger.Printf("=== ApplyQuick: %s ===", filePath)
	req := ApplyRequest{FilePath: filePath, Artist: artist, Title: title, Extras: extras}
	results := a.processApply([]ApplyRequest{req}, "", true)
	if len(results) > 0 {
		return results[0]
	}
	return ApplyResult{FilePath: filePath, Error: "no result"}
}

func (a *App) processApply(requests []ApplyRequest, outputDir string, overwrite bool) []ApplyResult {
	results := make([]ApplyResult, 0, len(requests))
	var historyEntries []HistoryEntry
	claimed := make(map[string]bool) // track paths claimed within this batch
	freed := make(map[string]bool)   // track source paths freed by renames

	for _, req := range requests {
		logger.Printf("processing: %s", filepath.Base(req.FilePath))

		// Deduplicate extras and strip from title to prevent accumulation
		req.Extras = deduplicateExtras(req.Extras)
		req.Title = stripExtrasFromTitle(req.Title, req.Extras)

		logger.Printf("  artist=%q title=%q extras=%q track=%d overwrite=%v",
			req.Artist, req.Title, req.Extras, req.Track, overwrite)

		// Read original tags for undo history
		origTags, _ := tagger.Read(req.FilePath)

		ext := filepath.Ext(req.FilePath)
		newFilename := buildNewFilename(req.Artist, req.Title, req.Extras, ext)
		if newFilename == "" {
			newFilename = filepath.Base(req.FilePath)
		}

		tagTitle := req.Title
		if req.Extras != "" {
			tagTitle = tagTitle + " (" + req.Extras + ")"
		}
		tags := tagger.Tags{
			Artist: req.Artist,
			Title:  tagTitle,
			Track:  req.Track,
		}

		if overwrite {
			// Verify file still exists (may have been renamed by earlier ApplyQuick)
			if _, err := os.Stat(req.FilePath); os.IsNotExist(err) {
				logger.Printf("  SKIP: file no longer exists (already renamed?)")
				results = append(results, ApplyResult{
					FilePath: req.FilePath, Error: "file no longer exists",
				})
				continue
			}

			// Write tags on the original
			if err := tagger.Write(req.FilePath, tags); err != nil {
				logger.Printf("  TAG WRITE ERROR: %v", err)
				results = append(results, ApplyResult{
					FilePath: req.FilePath, Error: fmt.Sprintf("tags failed: %v", err),
				})
				continue
			}

			// Rename the original
			destPath := filepath.Join(filepath.Dir(req.FilePath), newFilename)

			if destPath == req.FilePath {
				logger.Printf("  OK (overwrite, same name): %s", newFilename)
			} else if pathsEqual(destPath, req.FilePath) {
				// Only case changed — rename directly without uniquePath
				if err := os.Rename(req.FilePath, destPath); err != nil {
					logger.Printf("  RENAME ERROR: %v", err)
					results = append(results, ApplyResult{
						FilePath:    req.FilePath,
						NewFilename: filepath.Base(req.FilePath),
						Error:       fmt.Sprintf("rename failed (tags ok): %v", err),
					})
					continue
				}
				freed[strings.ToLower(req.FilePath)] = true
				logger.Printf("  OK (overwrite, case fix): %s", filepath.Base(destPath))
			} else {
				destPath = uniquePathWithClaimed(destPath, claimed, freed)
				if err := os.Rename(req.FilePath, destPath); err != nil {
					logger.Printf("  RENAME ERROR: %v", err)
					results = append(results, ApplyResult{
						FilePath:    req.FilePath,
						NewFilename: filepath.Base(req.FilePath),
						Error:       fmt.Sprintf("rename failed (tags ok): %v", err),
					})
					continue
				}
				freed[strings.ToLower(req.FilePath)] = true
				logger.Printf("  OK (overwrite): %s", filepath.Base(destPath))
			}
			claimed[strings.ToLower(destPath)] = true

			historyEntries = append(historyEntries, HistoryEntry{
				OriginalPath: req.FilePath,
				NewPath:      destPath,
				OriginalTags: origTags,
				NewTags:      tags,
			})
			results = append(results, ApplyResult{
				FilePath:    req.FilePath,
				NewPath:     destPath,
				NewFilename: filepath.Base(destPath),
				Success:     true,
			})
		} else {
			// Copy mode
			destPath := filepath.Join(outputDir, newFilename)
			destPath = uniquePathWithClaimed(destPath, claimed, nil)
			claimed[strings.ToLower(destPath)] = true

			logger.Printf("  copying to: %s", destPath)

			if err := copyFile(req.FilePath, destPath); err != nil {
				logger.Printf("  COPY ERROR: %v", err)
				results = append(results, ApplyResult{
					FilePath: req.FilePath, Error: fmt.Sprintf("copy failed: %v", err),
				})
				continue
			}

			if err := tagger.Write(destPath, tags); err != nil {
				logger.Printf("  TAG WRITE ERROR: %v", err)
				results = append(results, ApplyResult{
					FilePath:    req.FilePath,
					NewPath:     destPath,
					NewFilename: filepath.Base(destPath),
					Error:       fmt.Sprintf("tags failed (file copied ok): %v", err),
				})
				continue
			}

			logger.Printf("  OK (copy): %s", filepath.Base(destPath))
			historyEntries = append(historyEntries, HistoryEntry{
				OriginalPath: req.FilePath,
				NewPath:      destPath,
				OriginalTags: origTags,
				NewTags:      tags,
			})
			results = append(results, ApplyResult{
				FilePath:    req.FilePath,
				NewPath:     destPath,
				NewFilename: filepath.Base(destPath),
				Success:     true,
			})
		}
	}

	// Record history for undo
	mode := "gui-copy"
	if overwrite {
		mode = "gui-overwrite"
	}
	recordBatch(mode, historyEntries)

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	logger.Printf("=== Done: %d/%d successful ===", successCount, len(results))
	return results
}

// UndoLast reverts the most recent batch. Returns original file paths for UI rescan.
func (a *App) UndoLast() ([]string, error) {
	logger.Println("=== UndoLast called ===")
	paths, err := undoLastBatch()
	if err != nil {
		logger.Printf("undo error: %v", err)
		return nil, err
	}
	logger.Printf("undo complete: %d files reverted", len(paths))
	return paths, nil
}

// RedoLast re-applies the most recently undone batch. Returns new file paths for UI rescan.
func (a *App) RedoLast() ([]string, error) {
	logger.Println("=== RedoLast called ===")
	paths, err := redoLastBatch()
	if err != nil {
		logger.Printf("redo error: %v", err)
		return nil, err
	}
	logger.Printf("redo complete: %d files reapplied", len(paths))
	return paths, nil
}

// GetHistoryBatches returns recent history for display.
func (a *App) GetHistoryBatches() []HistoryBatch {
	h := loadHistory()
	return h.Batches
}

func (a *App) OpenOutputFolder(sourcePath string) {
	sourceDir := filepath.Dir(sourcePath)
	outputDir := filepath.Join(sourceDir, outputFolderName)
	logger.Printf("opening output folder: %s", outputDir)

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", outputDir)
	case "darwin":
		cmd = exec.Command("open", outputDir)
	default:
		cmd = exec.Command("xdg-open", outputDir)
	}
	if err := cmd.Start(); err != nil {
		logger.Printf("failed to open folder: %v", err)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// pathsEqual compares two paths case-insensitively on Windows,
// case-sensitively on other platforms.
func pathsEqual(a, b string) bool {
	if goruntime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func uniquePath(path string) string {
	return uniquePathWithClaimed(path, nil, nil)
}

func uniquePathWithClaimed(path string, claimed map[string]bool, freed map[string]bool) string {
	isClaimed := func(p string) bool {
		key := strings.ToLower(p)
		if freed != nil && freed[key] {
			return false // explicitly freed by rename in this batch
		}
		if claimed != nil && claimed[key] {
			return true
		}
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			return true
		}
		return false
	}

	if !isClaimed(path) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; i < 1000; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if !isClaimed(candidate) {
			return candidate
		}
	}
	return path
}

// stripExtrasFromTitle removes "(extras)" suffix from a title to prevent duplication
// when extras are appended separately. Case-insensitive to catch mixed-case edits.
func stripExtrasFromTitle(title, extras string) string {
	if extras == "" || title == "" {
		return title
	}
	suffix := " (" + extras + ")"
	for len(title) >= len(suffix) && strings.EqualFold(title[len(title)-len(suffix):], suffix) {
		title = strings.TrimSpace(title[:len(title)-len(suffix)])
	}
	return title
}

// deduplicateExtras removes duplicate comma-separated entries in extras.
func deduplicateExtras(extras string) string {
	if extras == "" {
		return ""
	}
	parts := strings.Split(extras, ", ")
	seen := make(map[string]bool)
	var unique []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}
	return strings.Join(unique, ", ")
}
