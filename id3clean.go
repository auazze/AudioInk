package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"go.senan.xyz/taglib"
)

// ID3 cleanup. taglib's Clear option rewrites the file's tags from a single
// property map, which:
//   - consolidates APE + ID3v1 + duplicate ID3v2 into one canonical ID3v2 block,
//   - drops non-standard / orphaned frames not in the property map.
//
// We preserve recognized standard tags (artist, title, album, date, genre,
// albumartist, …) and the embedded cover, and only strip a small denylist of
// junk keys plus URL-ish comments.
//
// Cleanup runs through processDestructive (copy → clean the copy), so Undo
// restores the FULL original bytes from the backup — no lossy tag-only revert.

// junkKeys are tag properties we always remove during cleanup.
var junkKeys = map[string]bool{
	"ENCODEDBY":       true,
	"ENCODERSETTINGS": true,
	"ENCODER":         true,
	"ITUNESADVISORY":  true,
	"ITUNNORM":        true,
	"ITUNSMPB":        true,
	"ITUNPGAP":        true,
}

func filterJunkTags(props map[string][]string) map[string][]string {
	out := make(map[string][]string, len(props))
	for k, v := range props {
		ku := strings.ToUpper(k)
		if junkKeys[ku] {
			continue
		}
		if strings.Contains(ku, "PODCAST") || strings.HasPrefix(ku, "WWW") || strings.Contains(ku, "URL") {
			continue
		}
		// Drop comment frames that are just a URL / spam line.
		if ku == "COMMENT" && len(v) > 0 && looksLikeURL(v[0]) {
			continue
		}
		out[k] = v
	}
	return out
}

func looksLikeURL(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.Contains(s, "www.")
}

// cleanTagsFile rewrites path's tags to a single canonical block, preserving
// the cover image.
func cleanTagsFile(path string) error {
	props, err := taglib.ReadTags(path)
	if err != nil {
		return err
	}
	img, _ := taglib.ReadImage(path)

	cleaned := filterJunkTags(props)
	if err := taglib.WriteTags(path, cleaned, taglib.Clear); err != nil {
		return err
	}

	// Re-embed the cover if the Clear pass dropped it.
	if len(img) > 0 {
		if cur, _ := taglib.ReadImage(path); len(cur) == 0 {
			_ = taglib.WriteImage(path, img)
		}
	}
	return nil
}

// CleanID3 strips junk/duplicate tag blocks from each file. overwrite chooses
// the save mode (in-place with backup vs copy to AudioInk/).
func (a *App) CleanID3(filePaths []string, overwrite bool) []ApplyResult {
	logger.Printf("=== CleanID3: %d files (overwrite=%v) ===", len(filePaths), overwrite)

	dreqs := make([]DestructiveRequest, 0, len(filePaths))
	for _, p := range filePaths {
		dreqs = append(dreqs, DestructiveRequest{FilePath: p, OutName: filepath.Base(p)})
	}
	op := func(ctx context.Context, src, dst string, onProgress func(float64)) error {
		if err := copyFile(src, dst); err != nil {
			return err
		}
		if err := cleanTagsFile(dst); err != nil {
			return err
		}
		if onProgress != nil {
			onProgress(1)
		}
		return nil
	}
	return a.processDestructive(dreqs, "clean", overwrite, op, nil)
}

// writeReplayGainTag stores the measured loudness adjustment as a standard
// ReplayGain track-gain tag. This is NON-destructive to the audio — a
// RG-aware player applies the gain at playback; the samples are untouched.
func writeReplayGainTag(path string, gainDB float64) error {
	val := fmt.Sprintf("%+.2f dB", gainDB)
	return taglib.WriteTags(path, map[string][]string{
		"REPLAYGAIN_TRACK_GAIN": {val},
	}, 0) // no Clear — merge into existing tags
}

// NormalizeFiles measures each file's loudness (EBU R128 via ffmpeg) and writes
// a ReplayGain track-gain tag so players can level the library on playback. The
// audio is never re-encoded (this is the non-destructive alternative to
// loudnorm). overwrite chooses the save mode; goes through the backup/undo path.
func (a *App) NormalizeFiles(filePaths []string, overwrite bool) []ApplyResult {
	logger.Printf("=== NormalizeFiles: %d files (overwrite=%v) ===", len(filePaths), overwrite)
	if a.audio == nil || !a.audio.HasFFmpeg() {
		return audioUnavailable(filePaths)
	}

	dreqs := make([]DestructiveRequest, 0, len(filePaths))
	for _, p := range filePaths {
		dreqs = append(dreqs, DestructiveRequest{FilePath: p, OutName: filepath.Base(p)})
	}
	op := func(ctx context.Context, src, dst string, onProgress func(float64)) error {
		gain, err := a.audio.ReplayGainDB(ctx, src)
		if err != nil {
			return err
		}
		if err := copyFile(src, dst); err != nil {
			return err
		}
		if err := writeReplayGainTag(dst, gain); err != nil {
			return err
		}
		logger.Printf("  normalize: %s → %+.2f dB ReplayGain", filepath.Base(src), gain)
		if onProgress != nil {
			onProgress(1)
		}
		return nil
	}
	return a.processDestructive(dreqs, "normalize", overwrite, op, nil)
}
