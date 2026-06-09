package audio

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
)

// SilenceEdges is the amount of near-digital silence at each end of a file,
// in seconds. Middle silence is deliberately ignored — we only ever trim edges.
type SilenceEdges struct {
	StartSec float64 `json:"startSec"` // trimmable silence at the beginning
	EndSec   float64 `json:"endSec"`   // trimmable silence at the end
}

// Any returns true if there's something worth trimming.
func (e SilenceEdges) Any() bool { return e.StartSec > 0.05 || e.EndSec > 0.05 }

const silenceThresholdDB = -50 // conservative: dead/digital silence only

// DetectSilence measures leading/trailing silence for the PREVIEW shown before
// trimming. It never modifies the file. The conservative threshold means an
// organic fade-in (which ramps through audible levels) is preserved — only the
// truly sub-threshold lead/tail is reported.
func (r *Runner) DetectSilence(ctx context.Context, path string) (SilenceEdges, error) {
	if !r.HasFFmpeg() {
		return SilenceEdges{}, fmt.Errorf("ffmpeg not available")
	}

	total := 0.0
	if specs, err := r.Probe(ctx, path); err == nil {
		total = specs.DurationSec
	}

	_, stderr, err := r.run(ctx, silenceTimeout, r.ffmpeg,
		"-hide_banner", "-nostats",
		"-i", path,
		"-map", "a:0",
		"-af", fmt.Sprintf("silencedetect=noise=%ddB:d=0.1", silenceThresholdDB),
		"-f", "null", nullSink(),
	)
	if err != nil {
		return SilenceEdges{}, fmt.Errorf("silencedetect %s: %w (%s)", path, err, tail(stderr))
	}
	return parseSilence(string(stderr), total), nil
}

// TrimSilence losslessly removes the leading/trailing silence given by edges,
// using stream-copy (`-c copy`) so the audio is NOT re-encoded (no quality
// loss). It is WYSIWYG with the DetectSilence preview and edges-only by
// construction. onProgress (may be nil) is called once with 1 on success —
// a copy-trim is effectively instant.
func (r *Runner) TrimSilence(ctx context.Context, src, dst string, edges SilenceEdges, onProgress func(float64)) error {
	if !r.HasFFmpeg() {
		return fmt.Errorf("ffmpeg not available")
	}

	total := 0.0
	if specs, err := r.Probe(ctx, src); err == nil {
		total = specs.DurationSec
	}
	keep := total - edges.StartSec - edges.EndSec
	if total <= 0 || keep <= 0 {
		return fmt.Errorf("trim would remove the entire file (duration=%.2f keep=%.2f)", total, keep)
	}

	args := []string{
		"-hide_banner", "-nostats", "-y",
		"-ss", ftoa(edges.StartSec),
		"-i", src,
		"-t", ftoa(keep),
		"-map", "a:0",
		"-map_metadata", "0",
		"-c", "copy",
		dst,
	}
	_, stderr, err := r.run(ctx, encodeTimeout, r.ffmpeg, args...)
	if err != nil {
		return fmt.Errorf("trim %s: %w (%s)", src, err, tail(stderr))
	}
	if onProgress != nil {
		onProgress(1)
	}
	return nil
}

var (
	reSilenceStart = regexp.MustCompile(`silence_start:\s*(-?[\d.]+)`)
	reSilenceEnd   = regexp.MustCompile(`silence_end:\s*(-?[\d.]+)`)
)

// parseSilence turns silencedetect stderr + total duration into edge amounts.
// Pure (no exec) for unit testing.
//
// silencedetect emits `silence_start: X` then later `silence_end: Y`. A region
// open at EOF (no silence_end) runs to `total`. We report:
//   - StartSec: the end of a region that begins at ~0 (leading silence)
//   - EndSec:   total minus the start of a region that ends at ~total (trailing)
func parseSilence(stderr string, total float64) SilenceEdges {
	const eps = 0.05

	type region struct{ start, end float64 }
	var regions []region
	var open float64 = -1
	haveOpen := false

	// Build a time-ordered event list by interleaving start/end matches via
	// their byte offsets in the stderr text.
	type ev struct {
		pos   int
		isEnd bool
		t     float64
	}
	var evs []ev
	for _, m := range reSilenceStart.FindAllStringSubmatchIndex(stderr, -1) {
		evs = append(evs, ev{pos: m[0], isEnd: false, t: mustFloat(stderr[m[2]:m[3]])})
	}
	for _, m := range reSilenceEnd.FindAllStringSubmatchIndex(stderr, -1) {
		evs = append(evs, ev{pos: m[0], isEnd: true, t: mustFloat(stderr[m[2]:m[3]])})
	}
	// sort by position
	for i := 1; i < len(evs); i++ {
		for j := i; j > 0 && evs[j-1].pos > evs[j].pos; j-- {
			evs[j-1], evs[j] = evs[j], evs[j-1]
		}
	}
	for _, e := range evs {
		if e.isEnd {
			if haveOpen {
				regions = append(regions, region{start: open, end: e.t})
				haveOpen = false
			}
		} else {
			open = e.t
			haveOpen = true
		}
	}
	if haveOpen && total > 0 {
		regions = append(regions, region{start: open, end: total})
	}

	var edges SilenceEdges
	for _, rg := range regions {
		if rg.start <= eps {
			if rg.end > edges.StartSec {
				edges.StartSec = rg.end
			}
		}
		if total > 0 && rg.end >= total-eps {
			if trail := total - rg.start; trail > edges.EndSec {
				edges.EndSec = trail
			}
		}
	}
	return edges
}

func ftoa(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}
