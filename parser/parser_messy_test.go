package parser

import (
	"reflect"
	"testing"
)

// TestMessyRealWorld covers download-site prefixes, yt-dlp ID suffixes,
// YouTube-converter junk, CJK punctuation, and quoted-title filenames.
func TestMessyRealWorld(t *testing.T) {
	type expect struct {
		artist string
		title  string
		track  int
		extras string
	}
	cases := []struct {
		input string
		want  expect
	}{
		// === Download-site prefixes stripped ===
		{"y2mate.com - Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"[Mp3Juices.cc] Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"www.NewAlbumReleases.net_Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"SPOTDOWNLOADER.COM - Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"yt5s.com-Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		// Artist names that merely look like domains survive (.am not a junk TLD)
		{"will.i.am - Scream & Shout.mp3", expect{"will.i.am", "Scream & Shout", 0, ""}},

		// === yt-dlp YouTube ID suffix stripped ===
		{"Rick Astley - Never Gonna Give You Up [dQw4w9WgXcQ].mp3",
			expect{"Rick Astley", "Never Gonna Give You Up", 0, ""}},
		{"Artist - Title [a1B2c3D4e5F].mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title [abc-DEF_123].mp3", expect{"Artist", "Title", 0, ""}},
		// Real extras of similar length survive (no digits / -_ inside)
		{"Artist - Title [Instrumental].mp3", expect{"Artist", "Title", 0, "Instrumental"}},
		{"Artist - Title [Remastered].mp3", expect{"Artist", "Title", 0, "Remastered"}},

		// === YouTube-converter junk extras stripped ===
		{"Artist - Title (Official Music Video) [4K].mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Lyrics).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Official Lyric Video).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Official Visualizer).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Visualizer).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Audio).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Official).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (MV).mp3", expect{"Artist", "Title", 0, ""}},
		// NB: "(M/V)" is untestable via Parse — "/" is a path separator on
		// every OS, so it can't appear in a real filename. The "m/v" junk
		// pattern still matters for CleanMetadata (tag values allow "/").
		{"Artist - Title (Official Video HD).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Color Coded Lyrics Han Rom Eng).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Sub Español).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Subtitulado).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (1080p).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (NCS Release).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Videoclip Oficial).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Clip Officiel).mp3", expect{"Artist", "Title", 0, ""}},
		// Meaningful extras still kept
		{"Artist - Title (Sub Focus Remix).mp3", expect{"Artist", "Title", 0, "Sub Focus Remix"}},
		{"Artist - Title (Slowed + Reverb).mp3", expect{"Artist", "Title", 0, "Slowed + Reverb"}},

		// === Trailing junk words WITHOUT brackets stripped ===
		{"Artist - Title Official Video.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title official audio.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title Official Music Video.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title Lyrics.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title HD.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title HQ.mp3", expect{"Artist", "Title", 0, ""}},
		// Real titles ending in similar words survive
		{"Lana Del Rey - Video Games.mp3", expect{"Lana Del Rey", "Video Games", 0, ""}},
		{"Artist - Radio Video.mp3", expect{"Artist", "Radio Video", 0, ""}},

		// === Leading bracketed junk stripped (【MV】, [FREE DOWNLOAD], etc.) ===
		{"【MV】Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"[Official Video] Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"(Free Download) Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},

		// === Full-width / CJK punctuation normalized ===
		{"Artist－Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title（Remix）.mp3", expect{"Artist", "Title", 0, "Remix"}},
		{"Artist - Title【Official Video】.mp3", expect{"Artist", "Title", 0, ""}},

		// === Quoted titles: «», “”, 「」, 『』 ===
		{"Кино «Группа крови».mp3", expect{"Кино", "Группа Крови", 0, ""}},
		{"Кино - «Группа крови».mp3", expect{"Кино", "Группа Крови", 0, ""}},
		{"米津玄師「Lemon」.mp3", expect{"米津玄師", "Lemon", 0, ""}},
		{"Artist “Title”.mp3", expect{"Artist", "Title", 0, ""}},
		{"AC-DC «Back In Black».mp3", expect{"AC-DC", "Back In Black", 0, ""}},

		// === Exotic dash variants normalized ===
		{"Artist—Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist–Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist−Title.mp3", expect{"Artist", "Title", 0, ""}}, // U+2212 minus

		// === (with X) treated as featured ===
		{"Artist - Title (with Someone).mp3", expect{"Artist", "Title", 0, ""}},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := Parse(tc.input)
			if got.Artist != tc.want.artist {
				t.Errorf("Artist: got %q, want %q", got.Artist, tc.want.artist)
			}
			if got.Title != tc.want.title {
				t.Errorf("Title: got %q, want %q", got.Title, tc.want.title)
			}
			if got.Track != tc.want.track {
				t.Errorf("Track: got %d, want %d", got.Track, tc.want.track)
			}
			if got.Extras != tc.want.extras {
				t.Errorf("Extras: got %q, want %q", got.Extras, tc.want.extras)
			}
		})
	}
}

func TestWithInParensIsFeatured(t *testing.T) {
	got := Parse("Tomoya Ohtani - Break Through It All (with Kellin Quinn).mp3")
	want := []string{"Kellin Quinn"}
	if !reflect.DeepEqual(got.FeaturedArtists, want) {
		t.Errorf("FeaturedArtists: got %v, want %v", got.FeaturedArtists, want)
	}
	if got.Title != "Break Through It All" {
		t.Errorf("Title: got %q", got.Title)
	}
}

func TestFeaturedDeduplicated(t *testing.T) {
	got := Parse("Artist feat. Guest - Title (feat. Guest).mp3")
	want := []string{"Guest"}
	if !reflect.DeepEqual(got.FeaturedArtists, want) {
		t.Errorf("FeaturedArtists: got %v, want %v", got.FeaturedArtists, want)
	}
}

func TestDigitsOnlyArtistLowConfidence(t *testing.T) {
	got := Parse("2024 - Title.mp3")
	if got.Confidence != Low {
		t.Errorf("Confidence: got %v, want Low for digits-only artist", got.Confidence)
	}
}
