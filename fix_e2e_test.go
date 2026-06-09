package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"AudioInk/tagger"
)

// copyTestAudio copies tagger/testdata/silence.<ext> into a temp dir.
// We reuse the audio fixtures generated for tagger tests.
func copyTestAudio(t *testing.T, ext, newName string) string {
	t.Helper()
	src := filepath.Join("tagger", "testdata", "silence."+ext)
	dst := filepath.Join(t.TempDir(), newName)

	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open testdata: %v (run tagger generation first)", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy: %v", err)
	}
	return dst
}

// === buildNewFilename ===

func TestBuildNewFilename(t *testing.T) {
	cases := []struct {
		artist, title, extras, ext string
		want                       string
	}{
		{"Daft Punk", "Get Lucky", "", ".mp3", "Daft Punk - Get Lucky.mp3"},
		{"Queen", "Bohemian Rhapsody", "Live", ".flac", "Queen - Bohemian Rhapsody (Live).flac"},
		{"", "Just A Title", "", ".mp3", "Just A Title.mp3"},
		{"Artist", "", "", ".mp3", ""}, // no title — empty result
		{"", "", "", ".mp3", ""},       // both empty
		{"A", "B", "Remix, Live", ".wav", "A - B (Remix, Live).wav"},
		{"Кино", "Группа крови", "", ".mp3", "Кино - Группа крови.mp3"},
		// Forbidden Windows chars stripped from artist + title
		{"AC/DC", "T:itle?", "", ".mp3", "ACDC - Title.mp3"},
		{`A<B>`, `C"D"`, "", ".mp3", "AB - CD.mp3"},
		// Double spaces collapsed
		{"A  B", "C  D", "", ".mp3", "A B - C D.mp3"},
	}
	for _, tc := range cases {
		t.Run(tc.artist+"|"+tc.title, func(t *testing.T) {
			got := buildNewFilename(tc.artist, tc.title, tc.extras, tc.ext)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// === sanitizeFilename ===

func TestSanitizeFilename_ForbiddenCharsStripped(t *testing.T) {
	// Windows: <>:"/\|?* are all illegal in filenames.
	in := `<>:"/\|?*safe<>:"/\|?*`
	got := sanitizeFilename(in)
	if got != "safe" {
		t.Errorf("got %q, want %q", got, "safe")
	}
}

func TestSanitizeFilename_PreservesUnicode(t *testing.T) {
	cases := []string{
		"Кино — Группа крови",
		"米津玄師 - Lemon",
		"P!nk - So What",  // ! is legal
		"Ke$ha - TiK ToK", // $ is legal
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got := sanitizeFilename(in)
			if got != in {
				t.Errorf("got %q, want %q (no chars stripped)", got, in)
			}
		})
	}
}

// === stripExtrasFromTitle ===

func TestStripExtrasFromTitle(t *testing.T) {
	cases := []struct {
		title, extras, want string
	}{
		{"Title (Remix)", "Remix", "Title"},
		{"Title (Live)", "Live", "Title"},
		{"Title", "Remix", "Title"},                 // no suffix to strip
		{"Title", "", "Title"},                      // no extras
		{"", "Remix", ""},                           // no title
		{"Title (Remix) (Remix)", "Remix", "Title"}, // multiple suffixes stripped
		{"Title (REMIX)", "Remix", "Title"},         // case-insensitive
		{"Title (remix)", "REMIX", "Title"},
		{"Title (Different)", "Remix", "Title (Different)"}, // mismatched extras kept
	}
	for _, tc := range cases {
		got := stripExtrasFromTitle(tc.title, tc.extras)
		if got != tc.want {
			t.Errorf("strip(%q, %q): got %q, want %q", tc.title, tc.extras, got, tc.want)
		}
	}
}

// === deduplicateExtras ===

func TestDeduplicateExtras(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Remix", "Remix"},
		{"", ""},
		{"Remix, Remix", "Remix"},
		{"Remix, Live", "Remix, Live"},
		{"Remix, Live, Remix, Acoustic, Live", "Remix, Live, Acoustic"},
		{"  Remix  ,  Live  ", "Remix, Live"}, // trimmed
	}
	for _, tc := range cases {
		got := deduplicateExtras(tc.in)
		if got != tc.want {
			t.Errorf("dedupe(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

// === uniquePath ===

func TestUniquePath_NoCollision(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fresh.mp3")
	got := uniquePath(path)
	if got != path {
		t.Errorf("got %q, want %q (no collision = unchanged)", got, path)
	}
}

func TestUniquePath_FileExists_AppendsSuffix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.mp3")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	got := uniquePath(path)
	want := filepath.Join(dir, "existing (2).mp3")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestUniquePath_MultipleCollisions_AppendsHigherSuffix(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "song.mp3")
	for _, p := range []string{base, filepath.Join(dir, "song (2).mp3"), filepath.Join(dir, "song (3).mp3")} {
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got := uniquePath(base)
	want := filepath.Join(dir, "song (4).mp3")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestUniquePathWithClaimed_RespectsBatchClaims(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "song.mp3")
	claimed := map[string]bool{
		strings.ToLower(path): true, // someone in this batch already grabbed it
	}
	got := uniquePathWithClaimed(path, claimed, nil)
	want := filepath.Join(dir, "song (2).mp3")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestUniquePathWithClaimed_FreedPathTakesPriority(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "song.mp3")
	// Pretend file exists but was just renamed away in this batch.
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	freed := map[string]bool{strings.ToLower(path): true}
	got := uniquePathWithClaimed(path, nil, freed)
	if got != path {
		t.Errorf("freed path should be reusable: got %q, want %q", got, path)
	}
}

// === pathsEqual ===

func TestPathsEqual(t *testing.T) {
	// On Windows, case-insensitive; elsewhere case-sensitive.
	// Both cases pass on both OSes for these inputs.
	if !pathsEqual("/a/b/c.mp3", "/a/b/c.mp3") {
		t.Error("identical paths should be equal")
	}
	if pathsEqual("/a/b/c.mp3", "/a/b/d.mp3") {
		t.Error("different paths should NOT be equal")
	}
}

// === fixOneFile end-to-end ===

func TestFixOneFile_FullFlow_MP3(t *testing.T) {
	initLogger()
	src := copyTestAudio(t, "mp3", "01. daft punk - get lucky.mp3")

	pf := parsedFile{
		filePath: src,
		artist:   "Daft Punk",
		title:    "Get Lucky",
		track:    1,
		filename: filepath.Base(src),
	}
	entry, err := fixOneFile(pf)
	if err != nil {
		t.Fatalf("fixOneFile: %v", err)
	}

	// File should have been renamed to clean format.
	expectedName := "Daft Punk - Get Lucky.mp3"
	if filepath.Base(entry.NewPath) != expectedName {
		t.Errorf("rename: got %q, want %q", filepath.Base(entry.NewPath), expectedName)
	}

	// Tags should be written.
	tags, err := tagger.Read(entry.NewPath)
	if err != nil {
		t.Fatalf("read tags after fix: %v", err)
	}
	if tags.Artist != "Daft Punk" || tags.Title != "Get Lucky" || tags.Track != 1 {
		t.Errorf("tags after fix: got %+v", tags)
	}
}

func TestFixOneFile_AllFormats(t *testing.T) {
	initLogger()
	formats := []string{"mp3", "flac", "ogg", "m4a", "wav", "wma", "opus"}
	for _, ext := range formats {
		t.Run(ext, func(t *testing.T) {
			src := copyTestAudio(t, ext, "messy name."+ext)
			pf := parsedFile{
				filePath: src,
				artist:   "Artist",
				title:    "Title",
				track:    5,
				filename: filepath.Base(src),
			}
			entry, err := fixOneFile(pf)
			if err != nil {
				t.Fatalf("fixOneFile %s: %v", ext, err)
			}
			tags, err := tagger.Read(entry.NewPath)
			if err != nil {
				t.Fatalf("read tags: %v", err)
			}
			if tags.Artist != "Artist" || tags.Title != "Title" || tags.Track != 5 {
				t.Errorf("%s: tags wrong: %+v", ext, tags)
			}
			if filepath.Base(entry.NewPath) != "Artist - Title."+ext {
				t.Errorf("%s: rename wrong: %q", ext, filepath.Base(entry.NewPath))
			}
		})
	}
}

func TestFixOneFile_WithExtras(t *testing.T) {
	initLogger()
	src := copyTestAudio(t, "mp3", "queen - bohemian rhapsody.mp3")
	pf := parsedFile{
		filePath: src,
		artist:   "Queen",
		title:    "Bohemian Rhapsody",
		extras:   "Live",
		filename: filepath.Base(src),
	}
	entry, err := fixOneFile(pf)
	if err != nil {
		t.Fatalf("fixOneFile: %v", err)
	}
	if filepath.Base(entry.NewPath) != "Queen - Bohemian Rhapsody (Live).mp3" {
		t.Errorf("rename: %q", filepath.Base(entry.NewPath))
	}
	tags, _ := tagger.Read(entry.NewPath)
	if tags.Title != "Bohemian Rhapsody (Live)" {
		t.Errorf("title tag should include extras: %q", tags.Title)
	}
}

func TestFixOneFile_AlreadyCorrectFilename_NoRenameNeeded(t *testing.T) {
	initLogger()
	src := copyTestAudio(t, "mp3", "Daft Punk - Get Lucky.mp3")
	pf := parsedFile{
		filePath: src,
		artist:   "Daft Punk",
		title:    "Get Lucky",
		filename: filepath.Base(src),
	}
	entry, err := fixOneFile(pf)
	if err != nil {
		t.Fatalf("fixOneFile: %v", err)
	}
	if entry.NewPath != src {
		t.Errorf("file should not have moved: src=%q new=%q", src, entry.NewPath)
	}
	// Tags should still be written
	tags, _ := tagger.Read(entry.NewPath)
	if tags.Artist != "Daft Punk" {
		t.Errorf("tags should be written even when no rename: %+v", tags)
	}
}

func TestFixOneFile_CollisionGetsUniquePath(t *testing.T) {
	initLogger()
	dir := t.TempDir()
	// Create the target name first to force collision
	target := filepath.Join(dir, "Daft Punk - Get Lucky.mp3")
	if err := os.WriteFile(target, []byte("squatter"), 0644); err != nil {
		t.Fatal(err)
	}

	// Now copy a real audio in with a different name
	src := filepath.Join(dir, "01 - get lucky.mp3")
	if err := copyFile(filepath.Join("tagger", "testdata", "silence.mp3"), src); err != nil {
		t.Fatal(err)
	}

	pf := parsedFile{
		filePath: src,
		artist:   "Daft Punk",
		title:    "Get Lucky",
		filename: filepath.Base(src),
	}
	entry, err := fixOneFile(pf)
	if err != nil {
		t.Fatalf("fixOneFile: %v", err)
	}
	expected := filepath.Join(dir, "Daft Punk - Get Lucky (2).mp3")
	if entry.NewPath != expected {
		t.Errorf("collision rename: got %q, want %q", entry.NewPath, expected)
	}
	// Original squatter should still exist
	if _, err := os.Stat(target); err != nil {
		t.Errorf("squatter file should be preserved: %v", err)
	}
}

func TestFixOneFile_CyrillicEndToEnd(t *testing.T) {
	initLogger()
	src := copyTestAudio(t, "mp3", "kino - gruppa krovi.mp3")
	pf := parsedFile{
		filePath: src,
		artist:   "Кино",
		title:    "Группа крови",
		filename: filepath.Base(src),
	}
	entry, err := fixOneFile(pf)
	if err != nil {
		t.Fatalf("fixOneFile: %v", err)
	}
	if filepath.Base(entry.NewPath) != "Кино - Группа крови.mp3" {
		t.Errorf("rename: %q", filepath.Base(entry.NewPath))
	}
	tags, _ := tagger.Read(entry.NewPath)
	if tags.Artist != "Кино" || tags.Title != "Группа крови" {
		t.Errorf("Cyrillic tags lost: %+v", tags)
	}
}

func TestFixOneFile_ForbiddenCharsInTitleSanitized(t *testing.T) {
	initLogger()
	// The renamer must strip Windows-forbidden chars; tag should preserve them.
	src := copyTestAudio(t, "mp3", "in.mp3")
	pf := parsedFile{
		filePath: src,
		artist:   "AC/DC",
		title:    "Question?",
		filename: filepath.Base(src),
	}
	entry, err := fixOneFile(pf)
	if err != nil {
		t.Fatalf("fixOneFile: %v", err)
	}
	// Filename: forbidden chars stripped
	if filepath.Base(entry.NewPath) != "ACDC - Question.mp3" {
		t.Errorf("filename should strip / and ?, got: %q", filepath.Base(entry.NewPath))
	}
	// Tag: original chars preserved
	tags, _ := tagger.Read(entry.NewPath)
	if tags.Artist != "AC/DC" || tags.Title != "Question?" {
		t.Errorf("tags should preserve forbidden chars: %+v", tags)
	}
}

// === fixPaths end-to-end ===

func TestFixPaths_TwoRealFiles(t *testing.T) {
	initLogger()
	dir := t.TempDir()

	// Two messy filenames, both with real audio data
	f1 := filepath.Join(dir, "01. daft punk - get lucky.mp3")
	f2 := filepath.Join(dir, "queen-bohemian rhapsody.flac")
	if err := copyFile(filepath.Join("tagger", "testdata", "silence.mp3"), f1); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join("tagger", "testdata", "silence.flac"), f2); err != nil {
		t.Fatal(err)
	}

	code := fixPaths([]string{f1, f2}, true)
	if code != 0 {
		t.Errorf("fixPaths returned %d, want 0", code)
	}

	// Check both got renamed cleanly
	want1 := filepath.Join(dir, "Daft Punk - Get Lucky.mp3")
	want2 := filepath.Join(dir, "Queen - Bohemian Rhapsody.flac")
	if _, err := os.Stat(want1); err != nil {
		t.Errorf("f1 not renamed correctly: %v", err)
	}
	if _, err := os.Stat(want2); err != nil {
		t.Errorf("f2 not renamed correctly: %v", err)
	}

	// And tags should be set
	t1, _ := tagger.Read(want1)
	if t1.Artist != "Daft Punk" || t1.Title != "Get Lucky" || t1.Track != 1 {
		t.Errorf("f1 tags: %+v", t1)
	}
	t2, _ := tagger.Read(want2)
	if t2.Artist != "Queen" || t2.Title != "Bohemian Rhapsody" {
		t.Errorf("f2 tags: %+v", t2)
	}
}

func TestFixPaths_MixedSupportedAndUnsupported(t *testing.T) {
	initLogger()
	dir := t.TempDir()

	mp3 := filepath.Join(dir, "01. artist - title.mp3")
	txt := filepath.Join(dir, "notes.txt")
	if err := copyFile(filepath.Join("tagger", "testdata", "silence.mp3"), mp3); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(txt, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	_ = fixPaths([]string{txt, mp3}, true)

	// txt should be untouched
	data, err := os.ReadFile(txt)
	if err != nil {
		t.Fatalf("txt should still exist: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("txt was modified: %q", string(data))
	}

	// mp3 should be renamed and tagged
	want := filepath.Join(dir, "Artist - Title.mp3")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("mp3 not renamed: %v", err)
	}
}
