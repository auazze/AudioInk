package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// Specs are the real audio-stream properties read from the file by ffprobe.
// taglib cannot provide these — it only reads tags.
type Specs struct {
	Codec       string  `json:"codec"`       // e.g. "mp3", "flac", "aac"
	BitrateKbps int     `json:"bitrateKbps"` // real bitrate from the container
	SampleRate  int     `json:"sampleRate"`  // Hz, e.g. 44100
	Channels    int     `json:"channels"`    // 1 mono, 2 stereo
	DurationSec float64 `json:"durationSec"`
	HasCover    bool    `json:"hasCover"`  // an attached_pic video stream is present
	Container   string  `json:"container"` // ffprobe format_name
}

// Probe reads audio specs via ffprobe. Read-only, fast (~tens of ms).
func (r *Runner) Probe(ctx context.Context, path string) (Specs, error) {
	if !r.HasFFprobe() {
		return Specs{}, fmt.Errorf("ffprobe not available")
	}
	stdout, stderr, err := r.run(ctx, probeTimeout, r.ffprobe,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	if err != nil {
		return Specs{}, fmt.Errorf("ffprobe %s: %w (%s)", path, err, tail(stderr))
	}
	return parseProbeJSON(stdout)
}

// ffprobe JSON shapes we care about. Numeric fields arrive as strings.
type probeOutput struct {
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		CodecType   string `json:"codec_type"`
		CodecName   string `json:"codec_name"`
		SampleRate  string `json:"sample_rate"`
		Channels    int    `json:"channels"`
		BitRate     string `json:"bit_rate"`
		Disposition struct {
			AttachedPic int `json:"attached_pic"`
		} `json:"disposition"`
	} `json:"streams"`
}

// parseProbeJSON decodes ffprobe output into Specs. Kept pure (no exec) so it
// is unit-testable with canned JSON.
func parseProbeJSON(data []byte) (Specs, error) {
	var po probeOutput
	if err := json.Unmarshal(data, &po); err != nil {
		return Specs{}, fmt.Errorf("parse ffprobe json: %w", err)
	}

	s := Specs{Container: po.Format.FormatName}
	s.DurationSec = atof(po.Format.Duration)

	// Prefer the format-level bitrate (whole container); fall back to the
	// audio stream's own bitrate if the container didn't report one.
	if bps := atoi(po.Format.BitRate); bps > 0 {
		s.BitrateKbps = bps / 1000
	}

	for _, st := range po.Streams {
		switch st.CodecType {
		case "audio":
			if s.Codec == "" {
				s.Codec = st.CodecName
				s.SampleRate = atoi(st.SampleRate)
				s.Channels = st.Channels
				if s.BitrateKbps == 0 {
					if bps := atoi(st.BitRate); bps > 0 {
						s.BitrateKbps = bps / 1000
					}
				}
			}
		case "video":
			// An album-art / cover is an "attached picture" video stream.
			if st.Disposition.AttachedPic == 1 {
				s.HasCover = true
			}
		}
	}

	if s.Codec == "" {
		return s, fmt.Errorf("no audio stream found")
	}
	return s, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// tail returns the last ~400 bytes of ffmpeg/ffprobe stderr for compact logs.
func tail(b []byte) string {
	const max = 400
	if len(b) > max {
		b = b[len(b)-max:]
	}
	return string(b)
}
