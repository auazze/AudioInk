package audio

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Health is the per-file library-hygiene verdict shown as a badge.
type Health struct {
	Status string   `json:"status"` // "ok" (green) | "warn" (yellow) | "bad" (red)
	Issues []string `json:"issues"` // human-readable, shown as tooltip
	Specs  Specs    `json:"specs"`
}

// expectedCodecs maps a file extension to the codec(s) ffprobe should report.
// A mismatch (e.g. a .mp3 that is really AAC) is a "warn".
var expectedCodecs = map[string][]string{
	".mp3":  {"mp3"},
	".flac": {"flac"},
	".ogg":  {"vorbis", "opus", "flac"},
	".opus": {"opus"},
	".m4a":  {"aac", "alac"},
	".wav":  {"pcm_s16le", "pcm_s24le", "pcm_s32le", "pcm_u8", "pcm_f32le"},
	".wma":  {"wmav1", "wmav2", "wmapro", "wmalossless"},
}

// Analyze produces a Health verdict. Cheap checks (specs, ext-vs-codec, cover)
// always run. Deep checks (silence, clipping, transcode/fake-320) only run when
// deep==true because they decode the whole file.
func (r *Runner) Analyze(ctx context.Context, path string, deep bool) (Health, error) {
	h := Health{Status: "ok"}

	specs, err := r.Probe(ctx, path)
	if err != nil {
		// Unreadable / corrupt container — the worst case.
		return Health{Status: "bad", Issues: []string{"unreadable or corrupt"}}, nil
	}
	h.Specs = specs

	// --- cheap checks ---
	ext := strings.ToLower(filepath.Ext(path))
	if want, ok := expectedCodecs[ext]; ok && specs.Codec != "" {
		if !contains(want, specs.Codec) {
			h.add("warn", fmt.Sprintf("extension says %s but audio is %s", ext, specs.Codec))
		}
	}
	if !specs.HasCover {
		h.note("no embedded cover") // informational only — does not change status
	}

	if !deep {
		return h, nil
	}

	// --- deep checks (decode required) ---
	// NOTE: we deliberately do NOT flag "clipping" from histogram_0db. Modern
	// loud masters legitimately sit thousands of samples at 0 dBFS, so that
	// counter false-positives on most commercial tracks. Real clipping needs
	// true-peak / consecutive-sample analysis, which is out of scope here.
	vd, derr := r.volumeDetect(ctx, path, "")
	if derr == nil {
		if vd.maxDB <= -50 {
			h.add("warn", "silent or near-silent")
		}
	} else if r.log != nil {
		r.log.Printf("audio: volumedetect failed for %s: %v", filepath.Base(path), derr)
	}

	// Fake-320 / transcode suspicion: a high-bitrate lossy file with no energy
	// above ~16 kHz was almost certainly upscaled from a lower bitrate.
	if isLossy(specs.Codec) && specs.BitrateKbps >= 256 {
		if hf, herr := r.volumeDetect(ctx, path, "highpass=f=16000,"); herr == nil {
			if hf.meanDB <= -78 {
				h.add("warn", fmt.Sprintf("possible transcode: %dkbps but no content above 16kHz", specs.BitrateKbps))
			}
		}
	}

	return h, nil
}

func (h *Health) add(sev, msg string) {
	h.Issues = append(h.Issues, msg)
	// bad beats warn beats ok
	if sev == "bad" || (sev == "warn" && h.Status == "ok") {
		h.Status = sev
	}
}

// note records an informational issue without changing the status.
func (h *Health) note(msg string) { h.Issues = append(h.Issues, msg) }

type volStats struct {
	maxDB   float64
	meanDB  float64
	clipped int // samples at 0 dBFS (histogram_0db)
}

var (
	reMaxVol  = regexp.MustCompile(`max_volume:\s*(-?[\d.]+) dB`)
	reMeanVol = regexp.MustCompile(`mean_volume:\s*(-?[\d.]+) dB`)
	reHist0   = regexp.MustCompile(`histogram_0db:\s*(\d+)`)
)

// volumeDetect runs ffmpeg's volumedetect filter (optionally behind a prefix
// filter chain like "highpass=f=16000,") and parses the stderr report.
func (r *Runner) volumeDetect(ctx context.Context, path, filterPrefix string) (volStats, error) {
	if !r.HasFFmpeg() {
		return volStats{}, fmt.Errorf("ffmpeg not available")
	}
	_, stderr, err := r.run(ctx, encodeTimeout, r.ffmpeg,
		"-hide_banner", "-nostats",
		"-i", path,
		"-map", "a:0",
		"-af", filterPrefix+"volumedetect",
		"-f", "null", nullSink(),
	)
	if err != nil {
		return volStats{}, fmt.Errorf("volumedetect: %w (%s)", err, tail(stderr))
	}
	return parseVolumeDetect(string(stderr)), nil
}

// parseVolumeDetect is pure (no exec) for unit testing.
func parseVolumeDetect(stderr string) volStats {
	var v volStats
	v.maxDB = -120
	v.meanDB = -120
	if m := reMaxVol.FindStringSubmatch(stderr); m != nil {
		v.maxDB = mustFloat(m[1])
	}
	if m := reMeanVol.FindStringSubmatch(stderr); m != nil {
		v.meanDB = mustFloat(m[1])
	}
	if m := reHist0.FindStringSubmatch(stderr); m != nil {
		v.clipped, _ = strconv.Atoi(m[1])
	}
	return v
}

func mustFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func isLossy(codec string) bool {
	switch codec {
	case "mp3", "aac", "vorbis", "opus", "wmav1", "wmav2", "wmapro":
		return true
	}
	return false
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
