package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"AudioInk/parser"
	"AudioInk/scanner"
	"AudioInk/tagger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var logger *log.Logger

func initLogger() {
	exe, _ := os.Executable()
	logPath := filepath.Join(filepath.Dir(exe), "audioink.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		// fallback: log next to working dir
		f, _ = os.OpenFile("audioink.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	}
	logger = log.New(f, "", log.Ltime)
	logger.Println("=== AudioInk started ===")
}

type App struct {
	ctx context.Context
}

func NewApp() *App {
	initLogger()
	return &App{}
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
	// Remove characters invalid in Windows filenames
	replacer := strings.NewReplacer(
		"<", "", ">", "", ":", "", "\"", "",
		"/", "", "\\", "", "|", "", "?", "", "*", "",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	// Collapse multiple spaces
	for strings.Contains(name, "  ") {
		name = strings.ReplaceAll(name, "  ", " ")
	}
	return name
}

func toFileResult(pr parser.ParseResult) FileResult {
	existing, err := tagger.Read(pr.FilePath)
	if err != nil {
		logger.Printf("  read tags error for %s: %v", pr.Filename, err)
	} else {
		logger.Printf("  existing tags: artist=%q title=%q", existing.Artist, existing.Title)
	}

	// Merge all artists with &
	allArtists := []string{}
	if pr.Artist != "" {
		allArtists = append(allArtists, pr.Artist)
	}
	allArtists = append(allArtists, pr.FeaturedArtists...)
	artist := strings.Join(allArtists, " & ")

	title := pr.Title

	// Extras like (Remix), (Live), (Acoustic) go at the end of filename
	extras := pr.Extras

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

func (a *App) ApplyTags(requests []ApplyRequest) []ApplyResult {
	logger.Printf("=== ApplyTags called with %d files ===", len(requests))

	if len(requests) == 0 {
		logger.Println("no files to process")
		return nil
	}

	// Determine output directory: _AudioInk_output/ next to source files
	sourceDir := filepath.Dir(requests[0].FilePath)
	outputDir := filepath.Join(sourceDir, "_AudioInk_output")

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

	results := make([]ApplyResult, 0, len(requests))

	for _, req := range requests {
		logger.Printf("processing: %s", filepath.Base(req.FilePath))
		logger.Printf("  artist=%q title=%q extras=%q track=%d", req.Artist, req.Title, req.Extras, req.Track)

		ext := filepath.Ext(req.FilePath)
		newFilename := buildNewFilename(req.Artist, req.Title, req.Extras, ext)
		if newFilename == "" {
			newFilename = filepath.Base(req.FilePath)
		}
		destPath := filepath.Join(outputDir, newFilename)

		// Avoid overwriting if file with same name exists
		destPath = uniquePath(destPath)

		logger.Printf("  copying to: %s", destPath)

		// Step 1: Copy file to output
		if err := copyFile(req.FilePath, destPath); err != nil {
			logger.Printf("  COPY ERROR: %v", err)
			results = append(results, ApplyResult{
				FilePath: req.FilePath, Error: fmt.Sprintf("copy failed: %v", err),
			})
			continue
		}

		// Step 2: Write tags on the copy
		// Title tag includes extras: "Song Title (Remix)"
		tagTitle := req.Title
		if req.Extras != "" {
			tagTitle = tagTitle + " (" + req.Extras + ")"
		}
		tags := tagger.Tags{
			Artist: req.Artist,
			Title:  tagTitle,
			Track:  req.Track,
		}
		if err := tagger.Write(destPath, tags); err != nil {
			logger.Printf("  TAG WRITE ERROR: %v", err)
			results = append(results, ApplyResult{
				FilePath: req.FilePath,
				NewPath:  destPath,
				NewFilename: filepath.Base(destPath),
				Error:    fmt.Sprintf("tags failed (file copied ok): %v", err),
			})
			continue
		}

		logger.Printf("  OK: %s", filepath.Base(destPath))
		results = append(results, ApplyResult{
			FilePath:    req.FilePath,
			NewPath:     destPath,
			NewFilename: filepath.Base(destPath),
			Success:     true,
		})
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	logger.Printf("=== Done: %d/%d successful, output in %s ===", successCount, len(results), outputDir)

	return results
}

func (a *App) OpenOutputFolder(sourcePath string) {
	sourceDir := filepath.Dir(sourcePath)
	outputDir := filepath.Join(sourceDir, "_AudioInk_output")
	logger.Printf("opening output folder: %s", outputDir)
	runtime.BrowserOpenURL(a.ctx, outputDir)
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

func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; i < 1000; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return path
}
