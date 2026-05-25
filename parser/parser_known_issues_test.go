package parser

import "testing"

// TestKnownIssues pins the parser's current (suboptimal) behavior for cases
// where the correct output is clear but fixing is out of scope or impossible.
// When any of these is fixed, the corresponding test will fail — forcing you
// to update the assertion to the correct value.
//
// Do not "fix" by changing the assertion — fix the parser.
func TestKnownIssues(t *testing.T) {
	type expect struct {
		artist string
		title  string
		track  int
	}
	cases := []struct {
		name         string
		input        string
		currentBuggy expect
		shouldBe     string
	}{
		{
			name:         "AC/DC — slash treated as path separator",
			input:        "AC/DC - Highway to Hell.mp3",
			currentBuggy: expect{"DC", "Highway To Hell", 0},
			shouldBe:     `Artist="AC/DC" — filepath.Base eats "AC/" because forward slash is a path separator. NOT FIXABLE: forward slash is forbidden in filenames on every major OS, so this input can't actually exist on disk. Real-world filename would be "AC-DC - Highway to Hell.mp3" or "ACDC...".`,
		},
		{
			name:         "All-caps K-pop band names lowercased (YOASOBI)",
			input:        "YOASOBI - Idol.mp3",
			currentBuggy: expect{"Yoasobi", "Idol", 0},
			shouldBe:     `Artist="YOASOBI" — TitleCase normalizes all-caps words. Many K-pop/J-pop groups are intentionally stylized all-caps (YOASOBI, TWICE, AESPA, NMIXX, RIIZE). Fix would require a whitelist or "preserve all-caps over N letters" heuristic, both with downsides (whitelist out-of-date, heuristic breaks "EMINEM - WITHOUT ME" normalization).`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse(tc.input)
			if got.Artist != tc.currentBuggy.artist {
				t.Errorf("Artist drift: got %q, pinned %q\n  → If you fixed the parser, the new value should be: %s",
					got.Artist, tc.currentBuggy.artist, tc.shouldBe)
			}
			if got.Title != tc.currentBuggy.title {
				t.Errorf("Title drift: got %q, pinned %q\n  → If you fixed the parser, the new value should be: %s",
					got.Title, tc.currentBuggy.title, tc.shouldBe)
			}
			if got.Track != tc.currentBuggy.track {
				t.Errorf("Track drift: got %d, pinned %d\n  → If you fixed the parser, the new value should be: %s",
					got.Track, tc.currentBuggy.track, tc.shouldBe)
			}
		})
	}
}
