package audio

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// These tests exercise the real ffmpeg/ffprobe/fpcalc binaries end-to-end.
// They skip automatically when the binaries aren't present (CI without the
// bundled toolchain), so `go test ./...` stays green everywhere.

func makeTestMP3(t *testing.T, r *Runner, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "tone.mp3")
	// 3s 440Hz sine, 0.8s leading silence, 1s trailing pad, CBR 320.
	cmd := exec.Command(r.ffmpeg, "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=3",
		"-af", "adelay=800|800,apad=pad_dur=1",
		"-c:a", "libmp3lame", "-b:a", "320k",
		path,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate test mp3: %v\n%s", err, out)
	}
	return path
}

func TestIntegrationProbeAndAnalyze(t *testing.T) {
	r := New(nil)
	if !r.HasFFmpeg() || !r.HasFFprobe() {
		t.Skip("ffmpeg/ffprobe not available")
	}
	dir := t.TempDir()
	src := makeTestMP3(t, r, dir)
	ctx := context.Background()

	specs, err := r.Probe(ctx, src)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if specs.Codec != "mp3" {
		t.Errorf("codec = %q, want mp3", specs.Codec)
	}
	if specs.SampleRate != 44100 {
		t.Errorf("sampleRate = %d, want 44100", specs.SampleRate)
	}
	if specs.BitrateKbps < 300 {
		t.Errorf("bitrate = %d, want ~320", specs.BitrateKbps)
	}

	h, err := r.Analyze(ctx, src, false)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if h.Specs.Codec != "mp3" {
		t.Errorf("analyze specs codec = %q", h.Specs.Codec)
	}
}

func TestIntegrationDetectSilence(t *testing.T) {
	r := New(nil)
	if !r.HasFFmpeg() {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	src := makeTestMP3(t, r, dir)

	edges, err := r.DetectSilence(context.Background(), src)
	if err != nil {
		t.Fatalf("detect silence: %v", err)
	}
	// ~0.8s leading, ~1s trailing.
	if edges.StartSec < 0.5 || edges.StartSec > 1.2 {
		t.Errorf("StartSec = %v, want ~0.8", edges.StartSec)
	}
	if edges.EndSec < 0.6 || edges.EndSec > 1.4 {
		t.Errorf("EndSec = %v, want ~1.0", edges.EndSec)
	}
}

func TestIntegrationConvertAndTrim(t *testing.T) {
	r := New(nil)
	if !r.HasFFmpeg() || !r.HasFFprobe() {
		t.Skip("ffmpeg/ffprobe not available")
	}
	dir := t.TempDir()
	src := makeTestMP3(t, r, dir)
	ctx := context.Background()

	// Convert mp3 → flac.
	flac := filepath.Join(dir, "out.flac")
	var lastPct float64
	if err := r.Convert(ctx, src, flac, ConvertOpts{TargetExt: ".flac"}, func(p float64) { lastPct = p }); err != nil {
		t.Fatalf("convert: %v", err)
	}
	if lastPct != 1 {
		t.Errorf("final progress = %v, want 1", lastPct)
	}
	fs, err := r.Probe(ctx, flac)
	if err != nil || fs.Codec != "flac" {
		t.Fatalf("converted file not flac: codec=%q err=%v", fs.Codec, err)
	}

	// Trim the silence off the original.
	trimmed := filepath.Join(dir, "trim.mp3")
	edges, _ := r.DetectSilence(ctx, src)
	if err := r.TrimSilence(ctx, src, trimmed, edges, nil); err != nil {
		t.Fatalf("trim: %v", err)
	}
	orig, _ := r.Probe(ctx, src)
	got, err := r.Probe(ctx, trimmed)
	if err != nil {
		t.Fatalf("probe trimmed: %v", err)
	}
	if got.DurationSec >= orig.DurationSec {
		t.Errorf("trimmed duration %.2f not shorter than original %.2f", got.DurationSec, orig.DurationSec)
	}
}

func TestIntegrationReplayGain(t *testing.T) {
	r := New(nil)
	if !r.HasFFmpeg() {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	src := makeTestMP3(t, r, dir)
	gain, err := r.ReplayGainDB(context.Background(), src)
	if err != nil {
		t.Fatalf("replaygain: %v", err)
	}
	// A quiet sine should suggest a positive gain toward -18 LUFS.
	if gain < -30 || gain > 30 {
		t.Errorf("gain = %v dB, out of sane range", gain)
	}
}

func TestIntegrationFingerprintAndDupes(t *testing.T) {
	r := New(nil)
	if !r.HasFFmpeg() || !r.HasFpcalc() {
		t.Skip("ffmpeg/fpcalc not available")
	}
	dir := t.TempDir()
	a := makeTestMP3(t, r, dir)
	ctx := context.Background()

	// Re-encode the same audio at a different bitrate → should still match.
	b := filepath.Join(dir, "tone-128.mp3")
	if err := r.Convert(ctx, a, b, ConvertOpts{TargetExt: ".mp3", Quality: "128k"}, nil); err != nil {
		t.Fatalf("convert variant: %v", err)
	}

	fpA, err := r.Fingerprint(ctx, a)
	if err != nil {
		t.Fatalf("fingerprint a: %v", err)
	}
	fpB, err := r.Fingerprint(ctx, b)
	if err != nil {
		t.Fatalf("fingerprint b: %v", err)
	}
	if len(fpA.Raw) == 0 || len(fpB.Raw) == 0 {
		t.Fatal("empty fingerprint")
	}
	groups := GroupDuplicates([]Fingerprint{fpA, fpB}, dupeSimilarityThreshold)
	if len(groups) != 1 || len(groups[0]) != 2 {
		t.Errorf("same song at 320 vs 128 should be 1 group of 2, got %v", groups)
	}
	_ = os.Remove(b)
}
