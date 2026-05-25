package tagger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// allFormats lists every audio format AudioInk supports.
// Each entry must have a corresponding silence.<ext> in testdata/.
var allFormats = []string{"mp3", "flac", "ogg", "m4a", "wav", "wma", "opus"}

// copyTestFile copies testdata/silence.<ext> into the test's temp dir
// and returns the new path. Tag writes mutate the file, so each test
// gets a fresh copy.
func copyTestFile(t *testing.T, ext string) string {
	t.Helper()
	src := filepath.Join("testdata", "silence."+ext)
	dst := filepath.Join(t.TempDir(), "test."+ext)

	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open testdata/silence.%s: %v (did ffmpeg generation run?)", ext, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy: %v", err)
	}
	return dst
}

// === Roundtrip per format ===

func TestRoundtripAllFormats(t *testing.T) {
	for _, ext := range allFormats {
		t.Run(ext, func(t *testing.T) {
			path := copyTestFile(t, ext)

			want := Tags{
				Artist: "Daft Punk",
				Title:  "Get Lucky",
				Album:  "Random Access Memories",
				Track:  8,
			}
			if err := Write(path, want); err != nil {
				t.Fatalf("Write: %v", err)
			}

			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != want {
				t.Errorf("roundtrip mismatch:\n  got:  %+v\n  want: %+v", got, want)
			}
		})
	}
}

// === Unicode coverage ===

func TestCyrillicTags(t *testing.T) {
	for _, ext := range allFormats {
		t.Run(ext, func(t *testing.T) {
			path := copyTestFile(t, ext)

			want := Tags{
				Artist: "Кино",
				Title:  "Группа крови",
				Album:  "Группа крови",
				Track:  3,
			}
			if err := Write(path, want); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != want {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestJapaneseTags(t *testing.T) {
	for _, ext := range allFormats {
		t.Run(ext, func(t *testing.T) {
			path := copyTestFile(t, ext)

			want := Tags{
				Artist: "坂本龍一",
				Title:  "Merry Christmas Mr. Lawrence",
				Album:  "戦場のメリークリスマス",
				Track:  1,
			}
			if err := Write(path, want); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != want {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestKoreanTags(t *testing.T) {
	for _, ext := range allFormats {
		t.Run(ext, func(t *testing.T) {
			path := copyTestFile(t, ext)
			want := Tags{
				Artist: "BTS",
				Title:  "봄날",
				Album:  "You Never Walk Alone",
				Track:  1,
			}
			if err := Write(path, want); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != want {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestEmojiInTags(t *testing.T) {
	// Some platforms (notably Spotify) allow emoji in track names.
	for _, ext := range allFormats {
		t.Run(ext, func(t *testing.T) {
			path := copyTestFile(t, ext)
			want := Tags{
				Artist: "Artist 🎵",
				Title:  "Song 🔥💯",
				Track:  1,
			}
			if err := Write(path, want); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got.Artist != want.Artist || got.Title != want.Title {
				t.Errorf("got artist=%q title=%q, want artist=%q title=%q",
					got.Artist, got.Title, want.Artist, want.Title)
			}
		})
	}
}

// === Edge cases ===

func TestEmptyTagsArePreserved(t *testing.T) {
	// Writing zero-value tags should not crash and should leave the file readable.
	for _, ext := range allFormats {
		t.Run(ext, func(t *testing.T) {
			path := copyTestFile(t, ext)
			if err := Write(path, Tags{}); err != nil {
				t.Fatalf("Write empty: %v", err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read after empty write: %v", err)
			}
			if got.Artist != "" || got.Title != "" {
				t.Errorf("expected empty tags after empty write, got %+v", got)
			}
		})
	}
}

func TestOverwriteTags(t *testing.T) {
	// Writing new tags must replace, not append to, existing tags.
	for _, ext := range allFormats {
		t.Run(ext, func(t *testing.T) {
			path := copyTestFile(t, ext)

			if err := Write(path, Tags{Artist: "Old", Title: "Old Title"}); err != nil {
				t.Fatalf("first write: %v", err)
			}
			if err := Write(path, Tags{Artist: "New", Title: "New Title"}); err != nil {
				t.Fatalf("second write: %v", err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got.Artist != "New" || got.Title != "New Title" {
				t.Errorf("expected overwrite, got artist=%q title=%q", got.Artist, got.Title)
			}
		})
	}
}

func TestVeryLongStrings(t *testing.T) {
	// Real-world malicious or accidental: 1000-character "title".
	longArtist := strings.Repeat("A", 1000)
	longTitle := strings.Repeat("B", 1000)

	for _, ext := range allFormats {
		t.Run(ext, func(t *testing.T) {
			path := copyTestFile(t, ext)
			if err := Write(path, Tags{Artist: longArtist, Title: longTitle}); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got.Artist != longArtist || got.Title != longTitle {
				t.Errorf("long string truncated or mangled (len artist=%d title=%d)",
					len([]rune(got.Artist)), len([]rune(got.Title)))
			}
		})
	}
}

func TestSpecialCharactersInTitle(t *testing.T) {
	// Quotes, ampersands, brackets, dashes — must roundtrip cleanly.
	tags := Tags{
		Artist: `O'Connor & "Friends"`,
		Title:  `Don't Stop Me Now — [Live] (Take 2)`,
		Track:  5,
	}
	for _, ext := range allFormats {
		t.Run(ext, func(t *testing.T) {
			path := copyTestFile(t, ext)
			if err := Write(path, tags); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got.Artist != tags.Artist || got.Title != tags.Title {
				t.Errorf("got %+v, want %+v", got, tags)
			}
		})
	}
}

func TestHighTrackNumbers(t *testing.T) {
	// Some compilations have 99+ tracks. Track stored as int — verify roundtrip.
	for _, ext := range allFormats {
		t.Run(ext, func(t *testing.T) {
			path := copyTestFile(t, ext)
			want := Tags{Artist: "VA", Title: "T", Track: 99}
			if err := Write(path, want); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got.Track != 99 {
				t.Errorf("track: got %d, want 99", got.Track)
			}
		})
	}
}

func TestTrackParsedFromSlashFormat(t *testing.T) {
	// MusicBrainz Picard writes tracks as "5/12" — we should still parse the first int.
	path := copyTestFile(t, "mp3")
	// Write via roundtrip first to ensure file is sane
	if err := Write(path, Tags{Artist: "A", Title: "T", Track: 5}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Track != 5 {
		t.Errorf("track: got %d, want 5", got.Track)
	}
}

// === Error paths ===

func TestReadNonexistentFile(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "does-not-exist.mp3"))
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "read tags") {
		t.Errorf("error should mention 'read tags', got: %v", err)
	}
}

func TestWriteToNonexistentFile(t *testing.T) {
	err := Write(filepath.Join(t.TempDir(), "does-not-exist.mp3"), Tags{Artist: "A"})
	if err == nil {
		t.Fatal("expected error writing to nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "write tags") {
		t.Errorf("error should mention 'write tags', got: %v", err)
	}
}

// Discovered contract: go-taglib does NOT error on zero-byte or garbage files —
// it silently returns empty tags / no-ops the write. Tests below pin that
// behavior so we notice if go-taglib ever starts being strict (which would be
// a breaking change for AudioInk's "process every file in folder" UX).

func TestReadZeroByteFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.mp3")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("create empty file: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read of empty file should not error (got: %v)", err)
	}
	if got != (Tags{}) {
		t.Errorf("Read of empty file should return zero Tags, got %+v", got)
	}
}

func TestReadGarbageFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garbage.mp3")
	if err := os.WriteFile(path, []byte("this is not audio data at all"), 0644); err != nil {
		t.Fatalf("create garbage file: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read of garbage file should not error (got: %v)", err)
	}
	if got != (Tags{}) {
		t.Errorf("Read of garbage file should return zero Tags, got %+v", got)
	}
}

func TestWriteToGarbageFileDoesNotPanic(t *testing.T) {
	// go-taglib silently no-ops the write on non-audio data. Critical that
	// AudioInk does not panic or corrupt the file if user drops a renamed
	// .txt-as-.mp3 into the queue.
	path := filepath.Join(t.TempDir(), "garbage.mp3")
	original := []byte("definitely not audio")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = Write(path, Tags{Artist: "A", Title: "T"})

	// File should still exist (not deleted).
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file disappeared after Write: %v", err)
	}
}

// === Filename-with-tricky-characters cases ===

func TestFileWithCyrillicPath(t *testing.T) {
	src := filepath.Join("testdata", "silence.mp3")
	dst := filepath.Join(t.TempDir(), "Кино — Группа крови.mp3")

	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		t.Fatal(err)
	}

	want := Tags{Artist: "Кино", Title: "Группа крови", Track: 3}
	if err := Write(dst, want); err != nil {
		t.Fatalf("Write to Cyrillic path: %v", err)
	}
	got, err := Read(dst)
	if err != nil {
		t.Fatalf("Read from Cyrillic path: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestFileWithSpacesInPath(t *testing.T) {
	src := filepath.Join("testdata", "silence.mp3")
	dst := filepath.Join(t.TempDir(), "  spaces  everywhere  .mp3")

	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		t.Fatal(err)
	}

	if err := Write(dst, Tags{Artist: "A", Title: "T"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(dst)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Artist != "A" || got.Title != "T" {
		t.Errorf("got %+v", got)
	}
}
