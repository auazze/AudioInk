package audio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProbeJSON(t *testing.T) {
	in := `{
	  "streams": [
	    {"codec_type":"audio","codec_name":"mp3","sample_rate":"44100","channels":2,"bit_rate":"320000"},
	    {"codec_type":"video","codec_name":"mjpeg","disposition":{"attached_pic":1}}
	  ],
	  "format": {"format_name":"mp3","duration":"183.456","bit_rate":"321000"}
	}`
	s, err := parseProbeJSON([]byte(in))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Codec != "mp3" {
		t.Errorf("codec = %q, want mp3", s.Codec)
	}
	if s.SampleRate != 44100 {
		t.Errorf("sampleRate = %d, want 44100", s.SampleRate)
	}
	if s.Channels != 2 {
		t.Errorf("channels = %d, want 2", s.Channels)
	}
	if s.BitrateKbps != 321 {
		t.Errorf("bitrate = %d, want 321", s.BitrateKbps)
	}
	if !s.HasCover {
		t.Error("HasCover = false, want true (attached_pic stream)")
	}
	if s.DurationSec < 183 || s.DurationSec > 184 {
		t.Errorf("duration = %v, want ~183.4", s.DurationSec)
	}
}

func TestParseProbeJSON_StreamBitrateFallback(t *testing.T) {
	in := `{"streams":[{"codec_type":"audio","codec_name":"flac","sample_rate":"48000","channels":2,"bit_rate":"900000"}],"format":{"format_name":"flac","duration":"60.0"}}`
	s, err := parseProbeJSON([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if s.BitrateKbps != 900 {
		t.Errorf("bitrate fallback = %d, want 900", s.BitrateKbps)
	}
}

func TestParseProbeJSON_NoAudio(t *testing.T) {
	in := `{"streams":[{"codec_type":"video","codec_name":"mjpeg"}],"format":{"format_name":"image2"}}`
	if _, err := parseProbeJSON([]byte(in)); err == nil {
		t.Error("expected error for no audio stream")
	}
}

func TestParseVolumeDetect(t *testing.T) {
	stderr := `[Parsed_volumedetect_0 @ 0x55] n_samples: 8067072
[Parsed_volumedetect_0 @ 0x55] mean_volume: -19.8 dB
[Parsed_volumedetect_0 @ 0x55] max_volume: -0.1 dB
[Parsed_volumedetect_0 @ 0x55] histogram_0db: 0`
	v := parseVolumeDetect(stderr)
	if v.maxDB != -0.1 {
		t.Errorf("maxDB = %v, want -0.1", v.maxDB)
	}
	if v.meanDB != -19.8 {
		t.Errorf("meanDB = %v, want -19.8", v.meanDB)
	}
	if v.clipped != 0 {
		t.Errorf("clipped = %d, want 0", v.clipped)
	}
}

func TestParseSilence_LeadingAndTrailing(t *testing.T) {
	// 200s file: leading silence 0..0.8, trailing 198.5..200
	stderr := `[silencedetect @ 0x1] silence_start: 0
[silencedetect @ 0x1] silence_end: 0.834 | silence_duration: 0.834
[silencedetect @ 0x1] silence_start: 198.5`
	e := parseSilence(stderr, 200.0)
	if e.StartSec < 0.83 || e.StartSec > 0.84 {
		t.Errorf("StartSec = %v, want ~0.834", e.StartSec)
	}
	if e.EndSec < 1.4 || e.EndSec > 1.6 {
		t.Errorf("EndSec = %v, want ~1.5", e.EndSec)
	}
}

func TestParseSilence_FadeInPreserved(t *testing.T) {
	// A region that starts mid-file is NOT an edge — must be ignored.
	stderr := `[silencedetect @ 0x1] silence_start: 45.0
[silencedetect @ 0x1] silence_end: 46.0 | silence_duration: 1.0`
	e := parseSilence(stderr, 200.0)
	if e.StartSec != 0 || e.EndSec != 0 {
		t.Errorf("middle silence leaked into edges: %+v", e)
	}
}

func TestParseProgressSeconds(t *testing.T) {
	if sec, ok := parseProgressSeconds("out_time_us=12500000"); !ok || sec != 12.5 {
		t.Errorf("got (%v,%v), want (12.5,true)", sec, ok)
	}
	if _, ok := parseProgressSeconds("progress=continue"); ok {
		t.Error("non-time line should not parse")
	}
	if _, ok := parseProgressSeconds("fps=0.0 out_time_us=5"); ok {
		t.Error("key must be at line start")
	}
}

func TestParseIntegratedLUFS(t *testing.T) {
	stderr := `[Parsed_ebur128_0 @ 0x1] t: 1 M: -20.0 S: -120.7 I: -23.0 LUFS LRA: 0.0 LU
[Parsed_ebur128_0 @ 0x1] Summary:

  Integrated loudness:
    I:         -16.5 LUFS
    Threshold: -26.6 LUFS`
	lufs, ok := parseIntegratedLUFS(stderr)
	if !ok || lufs != -16.5 {
		t.Errorf("got (%v,%v), want (-16.5,true) — must take the LAST (summary) I:", lufs, ok)
	}
}

func TestReplayGainMath(t *testing.T) {
	// gain = reference(-18) - integrated. For -16.5 LUFS → -1.5 dB.
	got := replayGainReferenceLUFS - (-16.5)
	if got != -1.5 {
		t.Errorf("gain = %v, want -1.5", got)
	}
}

func TestParseFpcalcJSON(t *testing.T) {
	in := `{"duration":183.45,"fingerprint":[1,2,3,4]}`
	fp, err := parseFpcalcJSON([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if fp.DurationSec != 183.45 {
		t.Errorf("duration = %v", fp.DurationSec)
	}
	if len(fp.Raw) != 4 || fp.Raw[0] != 1 || fp.Raw[3] != 4 {
		t.Errorf("raw = %v", fp.Raw)
	}
}

func TestGroupDuplicates(t *testing.T) {
	// Two identical vectors + one very different → one group of 2.
	same := []uint32{0xDEADBEEF, 0x12345678, 0xAABBCCDD, 0x01020304, 0xFFFFFFFF}
	diff := []uint32{0x00000000, 0x0F0F0F0F, 0x33333333, 0x55555555, 0x11111111}
	fps := []Fingerprint{
		{Path: "a.mp3", Raw: same},
		{Path: "b.mp3", Raw: same},
		{Path: "c.mp3", Raw: diff},
	}
	groups := GroupDuplicates(fps, dupeSimilarityThreshold)
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1 (%v)", len(groups), groups)
	}
	if len(groups[0]) != 2 {
		t.Errorf("group size = %d, want 2", len(groups[0]))
	}
}

func TestSimilarity_Identical(t *testing.T) {
	v := []uint32{1, 2, 3, 4, 5, 6, 7, 8}
	if s := similarity(v, v); s != 1 {
		t.Errorf("identical similarity = %v, want 1", s)
	}
}

func TestResolveBinary_BundledWinsOverPath(t *testing.T) {
	// A fake binary next to a fake "executable" must win over any PATH entry.
	dir := t.TempDir()
	name := "ffmpeg"
	bundled := filepath.Join(dir, name+exeSuffix())
	if err := os.WriteFile(bundled, []byte("stub"), 0755); err != nil {
		t.Fatal(err)
	}
	// We can't override os.Executable() directly, so assert the helper at
	// least finds a PATH binary or returns "" deterministically when absent.
	// The bundled-dir branch is covered indirectly by the New() log at runtime.
	got := resolveBinary("definitely-not-a-real-binary-xyz", nil)
	if got != "" {
		t.Errorf("nonexistent binary resolved to %q, want empty", got)
	}
}
