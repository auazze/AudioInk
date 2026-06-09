package audio

import (
	"bufio"
	"io"
)

// nullSink is the output target for analysis passes that discard audio
// (`-f null`). "-" (stdout) is the canonical cross-platform form.
func nullSink() string { return "-" }

// scanLines reads r line by line, invoking onLine for each. ffmpeg's
// `-progress pipe:1` emits one `key=value` per line, so a large buffer isn't
// needed, but we bump it anyway to be safe against pathological lines.
func scanLines(r io.Reader, onLine func(string)) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		onLine(sc.Text())
	}
}
