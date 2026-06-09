package audio

import (
	"context"
	"fmt"
	"regexp"
)

// replayGainReferenceLUFS is the ReplayGain 2.0 reference level. Track gain is
// the dB adjustment that would bring the file's integrated loudness to this.
const replayGainReferenceLUFS = -18.0

var reIntegratedLUFS = regexp.MustCompile(`I:\s*(-?[\d.]+)\s*LUFS`)

// ReplayGainDB measures the file's integrated loudness (EBU R128 via ffmpeg's
// ebur128 filter) and returns the ReplayGain track gain in dB. This is the
// value that would be written as a `replaygain_track_gain` TAG (non-destructive
// — the audio is never re-encoded; a RG-aware player applies the gain). It is
// NOT loudnorm.
//
// In the player it drives the A/B preview: gainLinear = 10^(gainDB/20).
func (r *Runner) ReplayGainDB(ctx context.Context, path string) (float64, error) {
	if !r.HasFFmpeg() {
		return 0, fmt.Errorf("ffmpeg not available")
	}
	_, stderr, err := r.run(ctx, encodeTimeout, r.ffmpeg,
		"-hide_banner", "-nostats",
		"-i", path,
		"-map", "a:0",
		"-af", "ebur128",
		"-f", "null", nullSink(),
	)
	if err != nil {
		return 0, fmt.Errorf("ebur128 %s: %w (%s)", path, err, tail(stderr))
	}
	lufs, ok := parseIntegratedLUFS(string(stderr))
	if !ok {
		return 0, fmt.Errorf("ebur128: integrated loudness not found")
	}
	return replayGainReferenceLUFS - lufs, nil
}

// parseIntegratedLUFS pulls the final (summary) integrated-loudness value from
// ebur128 stderr. The filter prints periodic `I:` lines and a final summary
// block; the LAST match is the integrated result. Pure for unit testing.
func parseIntegratedLUFS(stderr string) (float64, bool) {
	ms := reIntegratedLUFS.FindAllStringSubmatch(stderr, -1)
	if len(ms) == 0 {
		return 0, false
	}
	return mustFloat(ms[len(ms)-1][1]), true
}
