package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"AudioInk/audio"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// This file holds the Wails-bound methods for the FFmpeg-powered audio
// features, kept separate from app.go to respect the file-size limit. All
// ffmpeg/ffprobe/fpcalc work goes through a.audio (the audio.Runner chokepoint).

// AudioReady reports whether the bundled ffmpeg/ffprobe were found. The
// frontend uses this to gray out audio features when the binaries are missing.
func (a *App) AudioReady() bool {
	return a.audio != nil && a.audio.HasFFmpeg() && a.audio.HasFFprobe()
}

// ProbeHealth analyzes the given files and streams results back via events
// ("health:result" per file, "health:done" at the end). Returns immediately;
// the work runs in a goroutine so the UI stays responsive. deep=true enables
// the heavy decode-based checks (silence, clipping, transcode suspicion).
func (a *App) ProbeHealth(paths []string, deep bool) {
	if a.audio == nil || !a.audio.HasFFprobe() {
		logger.Println("ProbeHealth: ffprobe unavailable")
		runtime.EventsEmit(a.ctx, "health:done", map[string]any{"count": 0})
		return
	}
	go func() {
		ctx := context.Background()
		done := 0
		for _, p := range paths {
			h, err := a.audio.Analyze(ctx, p, deep)
			if err != nil {
				logger.Printf("ProbeHealth %s: %v", p, err)
				continue
			}
			if h.Status != "ok" {
				logger.Printf("  health %s: %s [%s]", h.Status, filepath.Base(p), strings.Join(h.Issues, "; "))
			}
			runtime.EventsEmit(a.ctx, "health:result", map[string]any{
				"filePath": p,
				"health":   h,
			})
			done++
		}
		runtime.EventsEmit(a.ctx, "health:done", map[string]any{"count": done})
		logger.Printf("ProbeHealth: analyzed %d/%d files (deep=%v)", done, len(paths), deep)
	}()
}

// ComputeReplayGain returns the ReplayGain track gain (dB) for the file — the
// value a RG-aware player would apply, used for the player's non-destructive
// A/B preview. The file is never modified.
func (a *App) ComputeReplayGain(filePath string) (float64, error) {
	if a.audio == nil || !a.audio.HasFFmpeg() {
		return 0, fmt.Errorf("ffmpeg unavailable")
	}
	gain, err := a.audio.ReplayGainDB(context.Background(), filePath)
	if err != nil {
		logger.Printf("ComputeReplayGain %s: %v", filePath, err)
		return 0, err
	}
	logger.Printf("ReplayGain %s: %.2f dB", filePath, gain)
	return gain, nil
}

// FindDuplicates fingerprints the given files locally (no network) and returns
// groups of paths that are the same recording. Emits "dupes:progress" while
// fingerprinting.
func (a *App) FindDuplicates(filePaths []string) ([][]string, error) {
	if a.audio == nil || !a.audio.HasFpcalc() {
		return nil, fmt.Errorf("fpcalc unavailable")
	}
	ctx := context.Background()
	fps := make([]audio.Fingerprint, 0, len(filePaths))
	for i, p := range filePaths {
		fp, err := a.audio.Fingerprint(ctx, p)
		if err != nil {
			logger.Printf("FindDuplicates fingerprint %s: %v", p, err)
			continue
		}
		fps = append(fps, fp)
		runtime.EventsEmit(a.ctx, "dupes:progress", map[string]any{
			"done": i + 1, "total": len(filePaths),
		})
	}
	groups := audio.GroupDuplicates(fps, 0.85)
	logger.Printf("FindDuplicates: %d files → %d duplicate group(s)", len(fps), len(groups))
	return groups, nil
}

// --- Destructive ops (convert / trim / repair) -----------------------------
// All route through processDestructive (backup.go), which mirrors the
// Copy-vs-Overwrite save modes and wires Undo.

type ConvertRequest struct {
	FilePath  string `json:"filePath"`
	TargetExt string `json:"targetExt"` // ".mp3", ".flac", ...
	Quality   string `json:"quality"`   // optional, e.g. "320k"
}

// ConvertFiles transcodes files to a target format. overwrite chooses the save
// mode (in-place with backup vs copy to AudioInk/). Emits "convert:progress".
func (a *App) ConvertFiles(reqs []ConvertRequest, overwrite bool) []ApplyResult {
	logger.Printf("=== ConvertFiles: %d files (overwrite=%v) ===", len(reqs), overwrite)
	if a.audio == nil || !a.audio.HasFFmpeg() {
		return audioUnavailable(pathsOfConvert(reqs))
	}

	dreqs := make([]DestructiveRequest, 0, len(reqs))
	optByPath := make(map[string]audio.ConvertOpts, len(reqs))
	for _, r := range reqs {
		ext := strings.ToLower(r.TargetExt)
		base := strings.TrimSuffix(filepath.Base(r.FilePath), filepath.Ext(r.FilePath))
		dreqs = append(dreqs, DestructiveRequest{FilePath: r.FilePath, OutName: base + ext})
		optByPath[r.FilePath] = audio.ConvertOpts{TargetExt: ext, Quality: r.Quality}
	}

	op := func(ctx context.Context, src, dst string, onProgress func(float64)) error {
		return a.audio.Convert(ctx, src, dst, optByPath[src], onProgress)
	}
	emit := func(filePath string, pct float64) {
		runtime.EventsEmit(a.ctx, "convert:progress", map[string]any{"filePath": filePath, "pct": pct})
	}
	results := a.processDestructive(dreqs, "convert", overwrite, op, emit)
	emitDone(a.ctx, "convert:done", results)
	return results
}

// DetectSilence measures trimmable leading/trailing silence for the preview.
func (a *App) DetectSilence(filePath string) (audio.SilenceEdges, error) {
	if a.audio == nil || !a.audio.HasFFmpeg() {
		return audio.SilenceEdges{}, fmt.Errorf("ffmpeg unavailable")
	}
	return a.audio.DetectSilence(context.Background(), filePath)
}

type TrimRequest struct {
	FilePath string             `json:"filePath"`
	Edges    audio.SilenceEdges `json:"edges"`
}

// TrimSilence losslessly removes the previewed edge silence from each file.
func (a *App) TrimSilence(reqs []TrimRequest, overwrite bool) []ApplyResult {
	logger.Printf("=== TrimSilence: %d files (overwrite=%v) ===", len(reqs), overwrite)
	if a.audio == nil || !a.audio.HasFFmpeg() {
		return audioUnavailable(pathsOfTrim(reqs))
	}

	dreqs := make([]DestructiveRequest, 0, len(reqs))
	edgesByPath := make(map[string]audio.SilenceEdges, len(reqs))
	for _, r := range reqs {
		dreqs = append(dreqs, DestructiveRequest{FilePath: r.FilePath, OutName: filepath.Base(r.FilePath)})
		edgesByPath[r.FilePath] = r.Edges
	}

	op := func(ctx context.Context, src, dst string, onProgress func(float64)) error {
		return a.audio.TrimSilence(ctx, src, dst, edgesByPath[src], onProgress)
	}
	emit := func(filePath string, pct float64) {
		runtime.EventsEmit(a.ctx, "trim:progress", map[string]any{"filePath": filePath, "pct": pct})
	}
	results := a.processDestructive(dreqs, "trim", overwrite, op, emit)
	emitDone(a.ctx, "trim:done", results)
	return results
}

// RepairFiles losslessly remuxes files to fix broken/missing VBR headers and
// containers.
func (a *App) RepairFiles(filePaths []string, overwrite bool) []ApplyResult {
	logger.Printf("=== RepairFiles: %d files (overwrite=%v) ===", len(filePaths), overwrite)
	if a.audio == nil || !a.audio.HasFFmpeg() {
		return audioUnavailable(filePaths)
	}

	dreqs := make([]DestructiveRequest, 0, len(filePaths))
	for _, p := range filePaths {
		dreqs = append(dreqs, DestructiveRequest{FilePath: p, OutName: filepath.Base(p)})
	}
	op := func(ctx context.Context, src, dst string, onProgress func(float64)) error {
		if err := a.audio.Remux(ctx, src, dst); err != nil {
			return err
		}
		if onProgress != nil {
			onProgress(1)
		}
		return nil
	}
	return a.processDestructive(dreqs, "repair", overwrite, op, nil)
}

// --- helpers ---

func audioUnavailable(paths []string) []ApplyResult {
	out := make([]ApplyResult, 0, len(paths))
	for _, p := range paths {
		out = append(out, ApplyResult{FilePath: p, Error: "ffmpeg unavailable"})
	}
	return out
}

func pathsOfConvert(reqs []ConvertRequest) []string {
	out := make([]string, len(reqs))
	for i, r := range reqs {
		out[i] = r.FilePath
	}
	return out
}

func pathsOfTrim(reqs []TrimRequest) []string {
	out := make([]string, len(reqs))
	for i, r := range reqs {
		out[i] = r.FilePath
	}
	return out
}

func emitDone(ctx context.Context, event string, results []ApplyResult) {
	for _, r := range results {
		runtime.EventsEmit(ctx, event, map[string]any{
			"filePath": r.FilePath,
			"success":  r.Success,
			"error":    r.Error,
			"newPath":  r.NewPath,
		})
	}
}
