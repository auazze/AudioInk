package parser

import (
	"testing"
)

func TestBasicArtistTitle(t *testing.T) {
	r := Parse("Daft Punk - Get Lucky.mp3")
	assertEq(t, r.Artist, "Daft Punk")
	assertEq(t, r.Title, "Get Lucky")
	assertConfidence(t, r.Confidence, High)
}

func TestTrackNumber(t *testing.T) {
	tests := []struct {
		input  string
		track  int
		artist string
		title  string
	}{
		{"01. Daft Punk - Get Lucky.mp3", 1, "Daft Punk", "Get Lucky"},
		{"3. Artist - Title.mp3", 3, "Artist", "Title"},
		{"12 - Artist - Title.flac", 12, "Artist", "Title"},
		{"05_Artist - Title.mp3", 5, "Artist", "Title"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input)
			if r.Track != tt.track {
				t.Errorf("track: got %d, want %d", r.Track, tt.track)
			}
			assertEq(t, r.Artist, tt.artist)
			assertEq(t, r.Title, tt.title)
		})
	}
}

func TestEmDashSeparator(t *testing.T) {
	r := Parse("Artist — Title.mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestEnDashSeparator(t *testing.T) {
	r := Parse("Artist – Title.mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestUnderscoreDashSeparator(t *testing.T) {
	r := Parse("Artist_-_Title.mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestExtrasParentheses(t *testing.T) {
	r := Parse("Daft Punk - Get Lucky (Remix).mp3")
	assertEq(t, r.Artist, "Daft Punk")
	assertEq(t, r.Title, "Get Lucky")
	assertEq(t, r.Extras, "Remix")
}

func TestExtrasBrackets(t *testing.T) {
	r := Parse("Artist - Title [Explicit].mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "Explicit")
}

func TestMultipleExtras(t *testing.T) {
	r := Parse("Artist - Title (Remastered 2009) [Explicit].mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	if r.Extras != "Remastered 2009, Explicit" {
		t.Errorf("extras: got %q, want %q", r.Extras, "Remastered 2009, Explicit")
	}
}

func TestFeaturedArtistInArtist(t *testing.T) {
	r := Parse("Drake feat. Rihanna - Take Care.mp3")
	assertEq(t, r.Artist, "Drake")
	assertEq(t, r.Title, "Take Care")
	if len(r.FeaturedArtists) != 1 || r.FeaturedArtists[0] != "Rihanna" {
		t.Errorf("featured: got %v, want [Rihanna]", r.FeaturedArtists)
	}
}

func TestFeaturedArtistInTitle(t *testing.T) {
	r := Parse("Drake - Take Care ft. Rihanna.mp3")
	assertEq(t, r.Artist, "Drake")
	assertEq(t, r.Title, "Take Care")
	if len(r.FeaturedArtists) != 1 || r.FeaturedArtists[0] != "Rihanna" {
		t.Errorf("featured: got %v, want [Rihanna]", r.FeaturedArtists)
	}
}

func TestMultipleFeaturedArtists(t *testing.T) {
	r := Parse("DJ Khaled feat. Drake & Lil Wayne - I'm The One.mp3")
	assertEq(t, r.Artist, "DJ Khaled")
	if len(r.FeaturedArtists) != 2 {
		t.Errorf("featured count: got %d, want 2 (%v)", len(r.FeaturedArtists), r.FeaturedArtists)
	}
}

func TestNoSeparator(t *testing.T) {
	r := Parse("some random filename.mp3")
	assertEq(t, r.Title, "Some Random Filename")
	assertEq(t, r.Artist, "")
	assertConfidence(t, r.Confidence, Low)
}

func TestCyrillicNames(t *testing.T) {
	r := Parse("Кино - Группа крови.mp3")
	assertEq(t, r.Artist, "Кино")
	assertEq(t, r.Title, "Группа Крови")
}

func TestTrackWithMultipleDashes(t *testing.T) {
	r := Parse("01. Twenty One Pilots - Stressed Out.mp3")
	assertEq(t, r.Artist, "Twenty One Pilots")
	assertEq(t, r.Title, "Stressed Out")
	if r.Track != 1 {
		t.Errorf("track: got %d, want 1", r.Track)
	}
}

func TestFilePath(t *testing.T) {
	r := Parse("/music/Artist - Title.mp3")
	assertEq(t, r.FilePath, "/music/Artist - Title.mp3")
	assertEq(t, r.Filename, "Artist - Title.mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestFlacExtension(t *testing.T) {
	r := Parse("Artist - Title.flac")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestLiveExtra(t *testing.T) {
	r := Parse("Queen - Bohemian Rhapsody (Live).flac")
	assertEq(t, r.Artist, "Queen")
	assertEq(t, r.Title, "Bohemian Rhapsody")
	assertEq(t, r.Extras, "Live")
}

func TestVsArtists(t *testing.T) {
	r := Parse("Artist1 vs. Artist2 - Battle Song.mp3")
	assertEq(t, r.Title, "Battle Song")
}

// === NEW: copy suffix stripping ===

func TestCopySuffixRussian(t *testing.T) {
	r := Parse("Tomoya Ohtani-Break Through It All (feat. Kellin Quinn) — копия.mp3")
	assertEq(t, r.Artist, "Tomoya Ohtani")
	assertEq(t, r.Title, "Break Through It All")
	if len(r.FeaturedArtists) == 0 || r.FeaturedArtists[0] != "Kellin Quinn" {
		t.Errorf("featured: got %v, want [Kellin Quinn]", r.FeaturedArtists)
	}
}

func TestCopySuffixEnglish(t *testing.T) {
	r := Parse("Artist - Title - Copy.mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestCopySuffixNumber(t *testing.T) {
	r := Parse("Artist - Title (2).mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

// === NEW: feat in parentheses ===

func TestFeatInParens(t *testing.T) {
	r := Parse("Artist - Title (feat. Someone).mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	if len(r.FeaturedArtists) != 1 || r.FeaturedArtists[0] != "Someone" {
		t.Errorf("featured: got %v, want [Someone]", r.FeaturedArtists)
	}
}

func TestFeatInParensMiddle(t *testing.T) {
	r := Parse("Tomoya Ohtani (feat. Kellin Quinn) - Break Through It All.mp3")
	assertEq(t, r.Artist, "Tomoya Ohtani")
	assertEq(t, r.Title, "Break Through It All")
	if len(r.FeaturedArtists) != 1 || r.FeaturedArtists[0] != "Kellin Quinn" {
		t.Errorf("featured: got %v, want [Kellin Quinn]", r.FeaturedArtists)
	}
}

func TestBareDashWithFeat(t *testing.T) {
	// No space-dash-space, only bare dash
	r := Parse("Tomoya Ohtani-Break Through It All.mp3")
	assertEq(t, r.Artist, "Tomoya Ohtani")
	assertEq(t, r.Title, "Break Through It All")
}

// === Trailing suffix stripping ===

func TestTrackSuffix(t *testing.T) {
	r := Parse("NF - GIVE ME A REASON_01.mp3")
	assertEq(t, r.Artist, "NF")
	assertEq(t, r.Title, "Give Me A Reason")
	if r.Track != 1 {
		t.Errorf("track: got %d, want 1", r.Track)
	}
}

func TestTrailingId(t *testing.T) {
	r := Parse("police-siren-21498.mp3")
	assertEq(t, r.Artist, "Police")
	assertEq(t, r.Title, "Siren")
}

func TestTrailingIdLong(t *testing.T) {
	r := Parse("sound-effect-83621.mp3")
	assertEq(t, r.Artist, "Sound")
	assertEq(t, r.Title, "Effect")
}

// === Title case ===

func TestTitleCaseAllCaps(t *testing.T) {
	r := Parse("EMINEM - WITHOUT ME.mp3")
	assertEq(t, r.Artist, "Eminem")
	assertEq(t, r.Title, "Without Me")
}

func TestTitleCaseAllLower(t *testing.T) {
	r := Parse("eminem - without me.mp3")
	assertEq(t, r.Artist, "Eminem")
	assertEq(t, r.Title, "Without Me")
}

func TestTitleCaseShortArtist(t *testing.T) {
	// Short names (<=3 chars) like "NF" should stay unchanged
	r := Parse("NF - The Search.mp3")
	assertEq(t, r.Artist, "NF")
	assertEq(t, r.Title, "The Search")
}

func TestTitleCaseMixedCaseUntouched(t *testing.T) {
	// Already proper case — don't touch
	r := Parse("Daft Punk - Get Lucky.mp3")
	assertEq(t, r.Artist, "Daft Punk")
	assertEq(t, r.Title, "Get Lucky")
}

func TestTitleCaseCyrillic(t *testing.T) {
	r := Parse("кино - группа крови.mp3")
	assertEq(t, r.Artist, "Кино")
	assertEq(t, r.Title, "Группа Крови")
}

// === Hyphenated names ===

func TestHyphenatedTitleNoSeparator(t *testing.T) {
	// All-lowercase hyphenated names: first segment = artist guess, rest = title
	r := Parse("lo-fi-hip-hop-beats.mp3")
	assertEq(t, r.Artist, "Lo")
	assertEq(t, r.Title, "Fi Hip Hop Beats")
	if r.Confidence != Low {
		t.Errorf("expected Low confidence, got %s", r.Confidence)
	}
}

func TestHyphenatedPlatformFilename(t *testing.T) {
	r := Parse("auazze-sold-to-the-wolves.mp3")
	assertEq(t, r.Artist, "Auazze")
	assertEq(t, r.Title, "Sold To The Wolves")
	if r.Confidence != Low {
		t.Errorf("expected Low confidence, got %s", r.Confidence)
	}
}

func TestHyphenMixedCasePreserved(t *testing.T) {
	// Mixed case hyphens like Wu-Tang should stay
	r := Parse("Wu-Tang - Gravel Pit.mp3")
	assertEq(t, r.Artist, "Wu-Tang")
	assertEq(t, r.Title, "Gravel Pit")
}

func TestTrackSuffixWithPrefix(t *testing.T) {
	// Prefix track takes priority over suffix
	r := Parse("05. Artist - Title_03.mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	if r.Track != 5 {
		t.Errorf("track: got %d, want 5 (prefix should win)", r.Track)
	}
}

// === VK / underscore-separated filenames ===

func TestVkFilename(t *testing.T) {
	r := Parse("aleksandr_novikov_roza_713893675_456239280_9dcc104f.mp3")
	assertEq(t, r.Artist, "Aleksandr Novikov")
	assertEq(t, r.Title, "Roza")
	assertConfidence(t, r.Confidence, Low)
}

func TestVkFilenameWithCopySuffix(t *testing.T) {
	r := Parse("aleksandr_novikov_roza_713893675_456239280_9dcc104f — копия.mp3")
	assertEq(t, r.Artist, "Aleksandr Novikov")
	assertEq(t, r.Title, "Roza")
}

func TestUnderscoreSeparatedSimple(t *testing.T) {
	// 4 words: split [2] artist + [2] title
	r := Parse("artist_name_song_title.mp3")
	assertEq(t, r.Artist, "Artist Name")
	assertEq(t, r.Title, "Song Title")
}

func TestUnderscoreWithTrailingNumbers(t *testing.T) {
	// 2 words after garbage strip: split [1] + [1]
	r := Parse("cool_song_12345678.mp3")
	assertEq(t, r.Artist, "Cool")
	assertEq(t, r.Title, "Song")
}

func TestUnderscoreSingleWord(t *testing.T) {
	// Only 1 word after cleanup — no split possible
	r := Parse("metallica_12345678.mp3")
	assertEq(t, r.Artist, "")
	assertEq(t, r.Title, "Metallica")
}

func TestUnderscoreDashSeparatorConversion(t *testing.T) {
	// _-_ should become " - " after underscore replacement
	r := Parse("Some_Artist_-_Some_Title.mp3")
	assertEq(t, r.Artist, "Some Artist")
	assertEq(t, r.Title, "Some Title")
}

// === Compound separators ===

func TestPlusDashPlusSeparator(t *testing.T) {
	r := Parse("Artist1+-+Title1.mp3")
	assertEq(t, r.Artist, "Artist1")
	assertEq(t, r.Title, "Title1")
}

func TestDoubleDashSeparator(t *testing.T) {
	r := Parse("Artist Name -- Song Title.mp3")
	assertEq(t, r.Artist, "Artist Name")
	assertEq(t, r.Title, "Song Title")
}

func TestTripleDashSeparator(t *testing.T) {
	r := Parse("Some Artist --- Some Song.flac")
	assertEq(t, r.Artist, "Some Artist")
	assertEq(t, r.Title, "Some Song")
}

func TestDoubleEqualsSeparator(t *testing.T) {
	r := Parse("MyArtist == MyTitle.ogg")
	assertEq(t, r.Artist, "MyArtist")
	assertEq(t, r.Title, "MyTitle")
}

// === Plus to ampersand ===

func TestPlusToAmpersandInArtist(t *testing.T) {
	r := Parse("Artist1 + Artist2 - Song Name.mp3")
	assertEq(t, r.Artist, "Artist1 & Artist2")
	assertEq(t, r.Title, "Song Name")
}

func TestMultiplePlusInArtist(t *testing.T) {
	r := Parse("A + B + C - Song.mp3")
	assertEq(t, r.Artist, "A & B & C")
	assertEq(t, r.Title, "Song")
}

// === Junk extra filtering ===

func TestJunkExtraBitrate(t *testing.T) {
	r := Parse("Artist - Title (320kbps).mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "")
}

func TestJunkExtraFormat(t *testing.T) {
	r := Parse("Artist - Title [FLAC].flac")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "")
}

func TestJunkExtraHQ(t *testing.T) {
	r := Parse("Artist - Title (HQ).mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "")
}

func TestJunkExtraOfficialVideo(t *testing.T) {
	r := Parse("Artist - Title (Official Video).mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "")
}

func TestJunkExtraButKeepRemix(t *testing.T) {
	// Remix is NOT junk, should be kept
	r := Parse("Artist - Title (Remix) [320kbps].mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "Remix")
}

func TestJunkExtraURL(t *testing.T) {
	r := Parse("Artist - Title (www.example.com).mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "")
}

func TestJunkExtraMusicVideo(t *testing.T) {
	r := Parse("Artist - Title [Music Video].mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "")
}

// === Curly braces ===

func TestCurlyBracesExtras(t *testing.T) {
	r := Parse("Artist - Title {Remix}.mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "Remix")
}

// === Track prefix with hash ===

func TestHashTrackPrefix(t *testing.T) {
	r := Parse("#05 Artist - Title.mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	if r.Track != 5 {
		t.Errorf("track: got %d, want 5", r.Track)
	}
}

// === Garbage character stripping ===

func TestGarbageCharsEdges(t *testing.T) {
	r := Parse("~~Artist - Title~~.mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestGarbageLeadingSpecialChars(t *testing.T) {
	r := Parse("~~~Artist - Title.mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestExclamationInTitle(t *testing.T) {
	// '!' should NOT be stripped — it's part of legitimate titles like SAD!
	r := Parse("XXXTENTACION - SAD!.mp3")
	assertEq(t, r.Artist, "Xxxtentacion")
	assertEq(t, r.Title, "Sad!")
}

func TestDollarInArtist(t *testing.T) {
	// '$' should NOT be stripped — used in real names like Ke$ha, A$AP Rocky
	r := Parse("Ke$ha - TiK ToK.mp3")
	assertEq(t, r.Artist, "Ke$ha")
	assertEq(t, r.Title, "TiK ToK")
}

// === Various audio extensions ===

func TestWavExtension(t *testing.T) {
	r := Parse("Artist - Title.wav")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestOggExtension(t *testing.T) {
	r := Parse("Artist - Title.ogg")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestM4aExtension(t *testing.T) {
	r := Parse("Artist - Title.m4a")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestOpusExtension(t *testing.T) {
	r := Parse("Artist - Title.opus")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

func TestWmaExtension(t *testing.T) {
	r := Parse("Artist - Title.wma")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
}

// === Complex real-world cases ===

func TestComplexMultipleAuthorsExtras(t *testing.T) {
	r := Parse("DJ Khaled + Drake + Lil Wayne - I'm The One (DJ Dog Remix).wav")
	assertEq(t, r.Artist, "DJ Khaled & Drake & Lil Wayne")
	assertEq(t, r.Title, "I'm The One")
	assertEq(t, r.Extras, "DJ Dog Remix")
}

func TestGarbageFilenameWithIds(t *testing.T) {
	r := Parse("cool_track_name_98765432_abcdef12.mp3")
	// Strip garbage IDs → "cool track name" (3 words) → [2]+[1] split
	assertEq(t, r.Artist, "Cool Track")
	assertEq(t, r.Title, "Name")
}

func TestFreeDownloadJunkExtra(t *testing.T) {
	r := Parse("Artist - Title (Free Download).mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "")
}

func TestKeepLiveAndAcousticExtras(t *testing.T) {
	r := Parse("Artist - Title (Live) (Acoustic).mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "Live, Acoustic")
}

func TestKeepRemasteredExtra(t *testing.T) {
	r := Parse("Queen - Bohemian Rhapsody (Remastered 2011).flac")
	assertEq(t, r.Artist, "Queen")
	assertEq(t, r.Title, "Bohemian Rhapsody")
	assertEq(t, r.Extras, "Remastered 2011")
}

func TestBareBitrate192(t *testing.T) {
	r := Parse("Artist - Title [192].mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "")
}

func TestLyricsVideoJunk(t *testing.T) {
	r := Parse("Artist - Title (Lyrics Video).mp3")
	assertEq(t, r.Artist, "Artist")
	assertEq(t, r.Title, "Title")
	assertEq(t, r.Extras, "")
}

func assertEq(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func assertConfidence(t *testing.T, got, want Confidence) {
	t.Helper()
	if got != want {
		t.Errorf("confidence: got %v, want %v", got, want)
	}
}
