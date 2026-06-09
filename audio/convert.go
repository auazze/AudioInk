package audio

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// ConvertOpts describes the target of a transcode. Quality is codec-specific:
// for MP3 a value like "320k" sets CBR 320, otherwise a sensible VBR default
// is used.
type ConvertOpts struct {
	TargetExt string // ".mp3", ".flac", ".m4a", ".ogg", ".opus", ".wav"
	Quality   string // optional, e.g. "320k"
}

// Convert transcodes src→dst, reporting progress 0..1 via onProgress (may be
// nil). Destructive at the call site (the caller decides Copy vs Overwrite);
// this function only writes dst.
func (r *Runner) Convert(ctx context.Context, src, dst string, o ConvertOpts, onProgress func(float64)) error {
	if !r.HasFFmpeg() {
		return fmt.Errorf("ffmpeg not available")
	}

	// Total duration drives the progress percentage.
	total := 0.0
	if specs, err := r.Probe(ctx, src); err == nil {
		total = specs.DurationSec
	}

	args := []string{
		"-hide_banner", "-nostats", "-y",
		"-i", src,
		"-map", "a:0", // audio only — drop video/cover streams during transcode
		"-map_metadata", "0", // carry tags over; taglib still owns tag correctness
	}
	args = append(args, codecArgs(o)...)
	args = append(args, "-progress", "pipe:1", dst)

	stderr, err := r.runStreaming(ctx, encodeTimeout, func(line string) {
		if onProgress == nil || total <= 0 {
			return
		}
		if sec, ok := parseProgressSeconds(line); ok {
			pct := sec / total
			if pct > 1 {
				pct = 1
			}
			onProgress(pct)
		}
	}, r.ffmpeg, args...)
	if err != nil {
		return fmt.Errorf("convert %s: %w (%s)", src, err, tail(stderr))
	}
	if onProgress != nil {
		onProgress(1)
	}
	return nil
}

// codecArgs maps a target extension + quality to ffmpeg encoder flags.
func codecArgs(o ConvertOpts) []string {
	switch strings.ToLower(o.TargetExt) {
	case ".mp3":
		if isBitrate(o.Quality) {
			return []string{"-c:a", "libmp3lame", "-b:a", o.Quality}
		}
		return []string{"-c:a", "libmp3lame", "-q:a", "2"} // VBR ~190 kbps
	case ".flac":
		return []string{"-c:a", "flac"}
	case ".m4a":
		q := o.Quality
		if !isBitrate(q) {
			q = "256k"
		}
		return []string{"-c:a", "aac", "-b:a", q}
	case ".ogg":
		return []string{"-c:a", "libvorbis", "-q:a", "6"}
	case ".opus":
		q := o.Quality
		if !isBitrate(q) {
			q = "192k"
		}
		return []string{"-c:a", "libopus", "-b:a", q}
	case ".wav":
		return []string{"-c:a", "pcm_s16le"}
	default:
		// Unknown target: copy stream (effectively a remux).
		return []string{"-c:a", "copy"}
	}
}

func isBitrate(q string) bool {
	return strings.HasSuffix(strings.ToLower(q), "k") && len(q) > 1
}

// parseProgressSeconds extracts the encoded position (seconds) from one
// ffmpeg `-progress` line. Pure for unit testing. Handles both `out_time_us`
// (microseconds) and `out_time_ms` (which ffmpeg confusingly emits in
// microseconds as well historically — we treat the us key as authoritative).
func parseProgressSeconds(line string) (float64, bool) {
	const key = "out_time_us="
	idx := strings.Index(line, key)
	if idx != 0 {
		return 0, false
	}
	val := strings.TrimSpace(line[len(key):])
	us, err := strconv.ParseFloat(val, 64)
	if err != nil || us < 0 {
		return 0, false
	}
	return us / 1e6, true
}
