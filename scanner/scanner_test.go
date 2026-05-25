package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// === IsSupported ===

func TestIsSupported_AllSupportedExtensions(t *testing.T) {
	supported := []string{"mp3", "flac", "ogg", "m4a", "wav", "wma", "opus"}
	for _, ext := range supported {
		t.Run(ext, func(t *testing.T) {
			if !IsSupported("song." + ext) {
				t.Errorf(".%s should be supported", ext)
			}
		})
	}
}

func TestIsSupported_CaseInsensitive(t *testing.T) {
	cases := []string{
		"song.MP3", "song.Mp3", "song.mP3", "song.mp3",
		"song.FLAC", "song.Flac",
		"song.OGG", "song.Ogg",
		"song.M4A", "song.m4A",
		"song.WAV", "song.Wav",
		"song.WMA", "song.Wma",
		"song.OPUS", "song.Opus",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			if !IsSupported(name) {
				t.Errorf("%q should be supported (case-insensitive)", name)
			}
		})
	}
}

func TestIsSupported_Unsupported(t *testing.T) {
	cases := []string{
		"document.txt",
		"image.png",
		"video.mp4", // not audio mp4
		"video.mkv",
		"archive.zip",
		"playlist.m3u",
		"playlist.pls",
		"data.json",
		"script.sh",
		"binary.exe",
		"midi.mid",
		"midi.midi",
		"audio.aiff", // explicitly not in supported list
		"audio.aac",  // explicitly not in supported list (only .m4a container)
		"audio.alac",
		"audio.ape",
		"audio.dts",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			if IsSupported(name) {
				t.Errorf("%q should NOT be supported", name)
			}
		})
	}
}

func TestIsSupported_NoExtension(t *testing.T) {
	cases := []string{"", "no_extension", "weird_filename_no_dot", "trailing_dot."}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			if IsSupported(name) {
				t.Errorf("%q should NOT be supported (no extension)", name)
			}
		})
	}
}

func TestIsSupported_FullPath(t *testing.T) {
	cases := []string{
		"/music/Daft Punk/song.mp3",
		`C:\Users\me\Music\song.mp3`,
		"./relative/path.flac",
		"../parent/path.opus",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			if !IsSupported(name) {
				t.Errorf("%q should be supported (full path)", name)
			}
		})
	}
}

func TestIsSupported_DotfileWithAudioExt(t *testing.T) {
	// A file literally named ".mp3" (no name, just extension)
	// — Go's filepath.Ext returns ".mp3" so it's considered supported.
	// This is consistent behavior; no special handling needed.
	if !IsSupported(".mp3") {
		t.Error("filename '.mp3' should be considered supported (Go treats it as extension-only)")
	}
}

// === ScanDirectory ===

func TestScanDirectory_Empty(t *testing.T) {
	dir := t.TempDir()
	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files in empty dir, got %d", len(files))
	}
}

func TestScanDirectory_OnlySupported(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.mp3", "b.flac", "c.ogg", "d.wav"} {
		mustCreate(t, filepath.Join(dir, name))
	}
	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(files) != 4 {
		t.Errorf("expected 4 files, got %d: %v", len(files), files)
	}
}

func TestScanDirectory_MixedFilesFilteredCorrectly(t *testing.T) {
	dir := t.TempDir()
	mustCreate(t, filepath.Join(dir, "song.mp3"))
	mustCreate(t, filepath.Join(dir, "readme.txt"))
	mustCreate(t, filepath.Join(dir, "cover.jpg"))
	mustCreate(t, filepath.Join(dir, "track.flac"))
	mustCreate(t, filepath.Join(dir, "playlist.m3u"))

	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 supported files, got %d: %v", len(files), files)
	}

	// Verify the right ones were picked
	got := basenames(files)
	sort.Strings(got)
	want := []string{"song.mp3", "track.flac"}
	if !equalSlices(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestScanDirectory_NestedRecursion(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "Artist1", "Album1"))
	mustMkdir(t, filepath.Join(dir, "Artist1", "Album2"))
	mustMkdir(t, filepath.Join(dir, "Artist2"))

	mustCreate(t, filepath.Join(dir, "top.mp3"))
	mustCreate(t, filepath.Join(dir, "Artist1", "Album1", "01.mp3"))
	mustCreate(t, filepath.Join(dir, "Artist1", "Album1", "02.flac"))
	mustCreate(t, filepath.Join(dir, "Artist1", "Album2", "song.ogg"))
	mustCreate(t, filepath.Join(dir, "Artist2", "track.wav"))
	mustCreate(t, filepath.Join(dir, "Artist2", "notes.txt")) // should be filtered

	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(files) != 5 {
		t.Errorf("expected 5 audio files, got %d: %v", len(files), files)
	}
}

func TestScanDirectory_DeepNesting(t *testing.T) {
	// 10 levels deep — make sure walker doesn't have weird depth limits.
	dir := t.TempDir()
	deep := dir
	for i := 0; i < 10; i++ {
		deep = filepath.Join(deep, "level")
	}
	mustMkdir(t, deep)
	mustCreate(t, filepath.Join(deep, "deep.mp3"))

	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 deeply-nested file, got %d: %v", len(files), files)
	}
}

func TestScanDirectory_CyrillicAndSpaceNames(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "Кино"))
	mustMkdir(t, filepath.Join(dir, "My Music Folder"))
	mustCreate(t, filepath.Join(dir, "Кино", "Группа крови.mp3"))
	mustCreate(t, filepath.Join(dir, "My Music Folder", "Song with spaces.flac"))

	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files (Cyrillic + spaces), got %d: %v", len(files), files)
	}
}

func TestScanDirectory_CaseInsensitiveExtensions(t *testing.T) {
	dir := t.TempDir()
	mustCreate(t, filepath.Join(dir, "loud.MP3"))
	mustCreate(t, filepath.Join(dir, "Mixed.Flac"))
	mustCreate(t, filepath.Join(dir, "shouted.OGG"))

	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 case-variant files, got %d: %v", len(files), files)
	}
}

func TestScanDirectory_HiddenFiles(t *testing.T) {
	// Hidden files (starting with .) ARE included — scanner doesn't filter them.
	// This is intentional: if a user manually drops a .Song.mp3, they want it processed.
	dir := t.TempDir()
	mustCreate(t, filepath.Join(dir, ".hidden_song.mp3"))
	mustCreate(t, filepath.Join(dir, "regular.mp3"))

	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected hidden + regular = 2 files, got %d: %v", len(files), files)
	}
}

func TestScanDirectory_NonexistentDirReturnsEmpty(t *testing.T) {
	// Discovered contract: ScanDirectory swallows all walk errors (including
	// "root does not exist") and returns an empty list. This is intentional
	// — drag-drop UX should never bubble OS-walk failures up; broken
	// symlinks / permission denied / missing root all silently yield empty.
	files, err := ScanDirectory(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Errorf("ScanDirectory of nonexistent dir should not error, got: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty result for nonexistent dir, got %v", files)
	}
}

func TestScanDirectory_FilePathInsteadOfDir(t *testing.T) {
	// filepath.Walk handles single files by walking with the file as root.
	// Verify behavior: if it's a supported audio file, it should be returned.
	dir := t.TempDir()
	file := filepath.Join(dir, "song.mp3")
	mustCreate(t, file)

	files, err := ScanDirectory(file)
	if err != nil {
		t.Fatalf("ScanDirectory on file: %v", err)
	}
	if len(files) != 1 || files[0] != file {
		t.Errorf("expected [%s], got %v", file, files)
	}
}

func TestScanDirectory_FilePathUnsupportedReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notes.txt")
	mustCreate(t, file)

	files, err := ScanDirectory(file)
	if err != nil {
		t.Fatalf("ScanDirectory on unsupported file: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty for unsupported file, got %v", files)
	}
}

func TestScanDirectory_AllSevenFormatsPresent(t *testing.T) {
	// Sanity: every supported format gets picked up in one scan.
	dir := t.TempDir()
	exts := []string{".mp3", ".flac", ".ogg", ".m4a", ".wav", ".wma", ".opus"}
	for i, ext := range exts {
		mustCreate(t, filepath.Join(dir, "song"+string(rune('a'+i))+ext))
	}

	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(files) != len(exts) {
		t.Errorf("expected %d files, got %d", len(exts), len(files))
	}
}

func TestScanDirectory_ManyFiles(t *testing.T) {
	// 500 files in one folder — confirm scanner doesn't choke on a typical library.
	dir := t.TempDir()
	const n = 500
	for i := 0; i < n; i++ {
		mustCreate(t, filepath.Join(dir, "track_"+itoa(i)+".mp3"))
	}
	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(files) != n {
		t.Errorf("expected %d files, got %d", n, len(files))
	}
}

func TestScanDirectory_PermissionDeniedSkipped(t *testing.T) {
	// On Unix, chmod 000 a subdir; scanner should skip rather than fail.
	// Windows ACL is too complex for a portable test — skip there.
	if runtime.GOOS == "windows" {
		t.Skip("permission test skipped on Windows")
	}
	dir := t.TempDir()
	denied := filepath.Join(dir, "denied")
	mustMkdir(t, denied)
	mustCreate(t, filepath.Join(denied, "hidden.mp3"))
	mustCreate(t, filepath.Join(dir, "visible.mp3"))

	if err := os.Chmod(denied, 0000); err != nil {
		t.Skipf("could not chmod: %v", err)
	}
	defer os.Chmod(denied, 0755) // cleanup so t.TempDir() can remove

	files, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("scanner should swallow permission errors, got: %v", err)
	}
	// Visible file should be present even if denied dir is skipped.
	found := false
	for _, f := range files {
		if filepath.Base(f) == "visible.mp3" {
			found = true
		}
	}
	if !found {
		t.Error("expected visible.mp3 in results despite denied sibling dir")
	}
}

// === FilterSupported ===

func TestFilterSupported_Empty(t *testing.T) {
	result := FilterSupported(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestFilterSupported_AllSupported(t *testing.T) {
	input := []string{"a.mp3", "b.flac", "c.opus"}
	result := FilterSupported(input)
	if len(result) != 3 {
		t.Errorf("expected 3, got %d: %v", len(result), result)
	}
}

func TestFilterSupported_NoneSupported(t *testing.T) {
	input := []string{"a.txt", "b.exe", "c.png"}
	result := FilterSupported(input)
	if len(result) != 0 {
		t.Errorf("expected 0, got %d: %v", len(result), result)
	}
}

func TestFilterSupported_Mixed(t *testing.T) {
	input := []string{
		"song.mp3",
		"readme.txt",
		"track.flac",
		"image.png",
		"audio.opus",
	}
	result := FilterSupported(input)
	if len(result) != 3 {
		t.Errorf("expected 3, got %d: %v", len(result), result)
	}
}

func TestFilterSupported_PreservesOrder(t *testing.T) {
	input := []string{"z.mp3", "a.txt", "m.flac", "b.txt", "c.ogg"}
	result := FilterSupported(input)
	want := []string{"z.mp3", "m.flac", "c.ogg"}
	if !equalSlices(result, want) {
		t.Errorf("order not preserved: got %v, want %v", result, want)
	}
}

// === helpers ===

func mustCreate(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func basenames(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = filepath.Base(p)
	}
	return out
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
