package audio

import (
	"context"
	"fmt"
)

// Remux rewrites the container losslessly (`-c copy`, no re-encode), which
// regenerates a correct VBR (Xing/Info) header and fixes broken/missing
// container structure. The audio stream itself is byte-identical. Used by the
// "Repair" feature for files the health report flags as corrupt or with bad
// headers (wrong duration / no seeking).
func (r *Runner) Remux(ctx context.Context, src, dst string) error {
	if !r.HasFFmpeg() {
		return fmt.Errorf("ffmpeg not available")
	}
	_, stderr, err := r.run(ctx, encodeTimeout, r.ffmpeg,
		"-hide_banner", "-nostats", "-y",
		"-i", src,
		"-map", "0", // keep all streams (audio + any cover)
		"-c", "copy",
		"-map_metadata", "0",
		dst,
	)
	if err != nil {
		return fmt.Errorf("remux %s: %w (%s)", src, err, tail(stderr))
	}
	return nil
}
