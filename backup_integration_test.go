package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"AudioInk/audio"
)

// These exercise processDestructive against the REAL ffmpeg, catching the
// class of bug where the temp file's extension matters (ffmpeg picks the muxer
// from it). The fake-op unit tests can't catch that. Skips without ffmpeg.

func genToneMP3(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "tone.mp3")
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-af", "adelay=800|800,apad=pad_dur=1",
		"-c:a", "libmp3lame", "-b:a", "320k", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gen tone: %v\n%s", err, out)
	}
	return path
}

func TestDestructiveRealConvertOverwrite(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	app := NewApp()
	dir := t.TempDir()
	src := genToneMP3(t, dir)

	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	op := func(ctx context.Context, s, d string, onP func(float64)) error {
		return app.audio.Convert(ctx, s, d, audio.ConvertOpts{TargetExt: ".flac"}, onP)
	}
	res := app.processDestructive(
		[]DestructiveRequest{{FilePath: src, OutName: base + ".flac"}},
		"convert", true, op, nil)

	if len(res) != 1 || !res[0].Success {
		t.Fatalf("convert overwrite failed: %+v", res)
	}
	if filepath.Ext(res[0].NewPath) != ".flac" {
		t.Errorf("expected .flac output, got %s", res[0].NewPath)
	}
	// Undo restores the original mp3 bytes.
	if _, err := undoLastBatch(); err != nil {
		t.Fatalf("undo: %v", err)
	}
}

func TestDestructiveRealTrimAndRepairOverwrite(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	app := NewApp()
	dir := t.TempDir()
	src := genToneMP3(t, dir)
	ctx := context.Background()

	// Trim (uses -c copy to a temp file — extension must be .mp3).
	edges, _ := app.audio.DetectSilence(ctx, src)
	trimOp := func(c context.Context, s, d string, onP func(float64)) error {
		return app.audio.TrimSilence(c, s, d, edges, onP)
	}
	res := app.processDestructive(
		[]DestructiveRequest{{FilePath: src, OutName: filepath.Base(src)}},
		"trim", true, trimOp, nil)
	if !res[0].Success {
		t.Fatalf("trim overwrite failed: %s", res[0].Error)
	}

	// Repair (remux -c copy).
	repairOp := func(c context.Context, s, d string, onP func(float64)) error {
		return app.audio.Remux(c, s, d)
	}
	res2 := app.processDestructive(
		[]DestructiveRequest{{FilePath: src, OutName: filepath.Base(src)}},
		"repair", true, repairOp, nil)
	if !res2[0].Success {
		t.Fatalf("repair overwrite failed: %s", res2[0].Error)
	}
}
