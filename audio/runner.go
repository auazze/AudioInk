// Package audio is the single chokepoint for every ffmpeg / ffprobe / fpcalc
// subprocess invocation in AudioInk. No other package shells out to these
// binaries. taglib stays the owner of metadata; this package only ever
// touches the audio stream (specs, transcode, loudness measurement, silence,
// remux, fingerprint).
//
// Binaries are resolved ONCE at startup. On Windows they are bundled next to
// AudioInk.exe by the NSIS installer, so we look there first (absolute path —
// critical because context-menu launches run from an arbitrary CWD). On
// macOS/Linux and in `wails dev` we fall back to PATH.
package audio

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"time"
)

// Per-operation timeouts. ffmpeg encodes can legitimately take minutes on a
// big lossless source, so the destructive ops get a generous ceiling; the
// read-only probes are short.
const (
	probeTimeout   = 15 * time.Second
	fpTimeout      = 30 * time.Second
	silenceTimeout = 30 * time.Second
	encodeTimeout  = 10 * time.Minute
)

// Runner holds the resolved absolute paths to the three external binaries.
// An empty string means "not found" — callers guard with the Has* helpers and
// surface a friendly error rather than crashing.
type Runner struct {
	ffmpeg  string
	ffprobe string
	fpcalc  string
	log     *log.Logger
}

// New resolves the binaries and logs where each one was found (or that it is
// missing). It never fails — a missing binary just disables the features that
// need it.
func New(logger *log.Logger) *Runner {
	r := &Runner{log: logger}
	r.ffmpeg = resolveBinary("ffmpeg", logger)
	r.ffprobe = resolveBinary("ffprobe", logger)
	r.fpcalc = resolveBinary("fpcalc", logger)
	return r
}

func (r *Runner) HasFFmpeg() bool  { return r.ffmpeg != "" }
func (r *Runner) HasFFprobe() bool { return r.ffprobe != "" }
func (r *Runner) HasFpcalc() bool  { return r.fpcalc != "" }

func exeSuffix() string {
	if goruntime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// resolveBinary returns the absolute path to a binary, preferring one bundled
// next to our own executable, then falling back to PATH.
func resolveBinary(name string, logger *log.Logger) string {
	if exe, err := os.Executable(); err == nil {
		cand := filepath.Join(filepath.Dir(exe), name+exeSuffix())
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			if logger != nil {
				logger.Printf("audio: %s resolved (bundled): %s", name, cand)
			}
			return cand
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		if logger != nil {
			logger.Printf("audio: %s resolved (PATH): %s", name, p)
		}
		return p
	}
	if logger != nil {
		logger.Printf("audio: %s NOT FOUND (bundled or PATH) — features needing it are disabled", name)
	}
	return ""
}

// run is the single exec primitive: runs bin with args under a timeout,
// capturing stdout and stderr separately. On a non-zero exit it returns an
// *ExitError-wrapped error along with whatever stderr was produced so callers
// can log the ffmpeg tail.
func (r *Runner) run(ctx context.Context, timeout time.Duration, bin string, args ...string) (stdout, stderr []byte, err error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, bin, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	hideWindow(cmd) // platform-unique: no console flash on Windows

	err = cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

// runStreaming runs bin and feeds each stdout line to onLine as it arrives.
// Used by Convert/TrimSilence to parse ffmpeg's `-progress pipe:1` output for
// live progress. stderr is collected for error reporting.
func (r *Runner) runStreaming(ctx context.Context, timeout time.Duration, onLine func(string), bin string, args ...string) (stderr []byte, err error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, bin, args...)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	hideWindow(cmd)

	pipe, perr := cmd.StdoutPipe()
	if perr != nil {
		return nil, perr
	}
	if serr := cmd.Start(); serr != nil {
		return nil, serr
	}

	scanLines(pipe, onLine)

	err = cmd.Wait()
	return errBuf.Bytes(), err
}
