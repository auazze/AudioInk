package parser

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestGenerateReport runs ALL crazy filenames through the parser and writes donetests.txt
func TestGenerateReport(t *testing.T) {
	cases := []string{
		// === BASIC SEPARATORS ===
		"Daft Punk - Get Lucky.mp3",
		"Artist — Title.mp3",
		"Artist – Title.mp3",
		"Artist_-_Title.mp3",
		"Artist+-+Title.mp3",
		"Artist -Title.mp3",
		"Artist- Title.mp3",
		"Artist--Title.mp3",
		"Artist---Title.mp3",
		"Artist==Title.mp3",
		"Artist===Title.flac",

		// === TRACK NUMBERS ===
		"01. Daft Punk - Get Lucky.mp3",
		"3. Artist - Title.mp3",
		"12 - Artist - Title.flac",
		"05_Artist - Title.mp3",
		"#05 Artist - Title.mp3",
		"#1. Song Name.mp3",
		"001 - Artist - Title.mp3",

		// === TRACK SUFFIX ===
		"NF - GIVE ME A REASON_01.mp3",
		"05. Artist - Title_03.mp3",
		"Artist - Title_99.mp3",

		// === FEATURED ARTISTS ===
		"Drake feat. Rihanna - Take Care.mp3",
		"Drake - Take Care ft. Rihanna.mp3",
		"DJ Khaled feat. Drake & Lil Wayne - I'm The One.mp3",
		"Artist - Title (feat. Someone).mp3",
		"Tomoya Ohtani (feat. Kellin Quinn) - Break Through It All.mp3",
		"Artist featuring B - Song.mp3",
		"A ft B - Song.mp3",

		// === PLUS TO AMPERSAND ===
		"Artist1 + Artist2 - Song Name.mp3",
		"A + B + C - Song.mp3",
		"DJ Snake + Lil Jon - Turn Down for What.mp3",

		// === EXTRAS (GOOD - KEEP) ===
		"Daft Punk - Get Lucky (Remix).mp3",
		"Artist - Title [Explicit].mp3",
		"Artist - Title (Remastered 2009) [Explicit].mp3",
		"Queen - Bohemian Rhapsody (Live).flac",
		"Artist - Title (Acoustic).mp3",
		"Artist - Title (Instrumental).mp3",
		"Artist - Title (Radio Edit).mp3",
		"Artist - Title (Extended Mix).mp3",
		"Artist - Title (Deluxe Edition).mp3",
		"Artist - Title (DJ Dog Remix).wav",
		"Artist - Title (Live) (Acoustic).mp3",
		"Artist - Title (Remastered 2011).flac",

		// === JUNK EXTRAS (STRIP) ===
		"Artist - Title (320kbps).mp3",
		"Artist - Title [FLAC].flac",
		"Artist - Title (HQ).mp3",
		"Artist - Title (Official Video).mp3",
		"Artist - Title (Remix) [320kbps].mp3",
		"Artist - Title (www.example.com).mp3",
		"Artist - Title [Music Video].mp3",
		"Artist - Title (128kb/s).mp3",
		"Artist - Title [192].mp3",
		"Artist - Title (Lyrics Video).mp3",
		"Artist - Title (Official Audio).mp3",
		"Artist - Title (Free Download).mp3",
		"Artist - Title (Audio Only).mp3",
		"Artist - Title [MP3].mp3",
		"Artist - Title (HD).mp3",
		"Artist - Title (Full Version).mp3",
		"Artist - Title [WAV].wav",
		"Artist - Title (LQ).mp3",
		"Artist - Title (CDQ).mp3",
		"Artist - Title [AAC].m4a",

		// === CURLY BRACES ===
		"Artist - Title {Remix}.mp3",
		"Artist - Title {Live}.mp3",
		"Artist - Title {Remastered 2020}.flac",

		// === COPY SUFFIXES ===
		"Tomoya Ohtani-Break Through It All (feat. Kellin Quinn) — копия.mp3",
		"Artist - Title - Copy.mp3",
		"Artist - Title (2).mp3",
		"Artist - Title — копия (3).mp3",
		"Artist - Title - Copy (2).mp3",
		"Artist - Title - copia.mp3",

		// === GARBAGE IDS / TRAILING NUMBERS ===
		"police-siren-21498.mp3",
		"sound-effect-83621.mp3",
		"aleksandr_novikov_roza_713893675_456239280_9dcc104f.mp3",
		"aleksandr_novikov_roza_713893675_456239280_9dcc104f — копия.mp3",
		"cool_track_name_98765432_abcdef12.mp3",
		"cool_song_12345678.mp3",
		"metallica_12345678.mp3",

		// === VK / UNDERSCORE NAMES ===
		"artist_name_song_title.mp3",
		"ivan_ivanov_pesnya_lubvi.mp3",
		"dj_snake_turn_down.mp3",
		"Some_Artist_-_Some_Title.mp3",

		// === ALL CAPS ===
		"EMINEM - WITHOUT ME.mp3",
		"METALLICA - ENTER SANDMAN.flac",
		"AC/DC - THUNDERSTRUCK.mp3",
		"DAFT PUNK - AROUND THE WORLD.mp3",

		// === ALL LOWERCASE ===
		"eminem - without me.mp3",
		"daft punk - get lucky.mp3",
		"queen - bohemian rhapsody.flac",

		// === CYRILLIC ===
		"Кино - Группа крови.mp3",
		"кино - группа крови.mp3",
		"КИНО - ГРУППА КРОВИ.mp3",
		"Виктор Цой - Звезда по имени Солнце.mp3",
		"ДДТ feat. Юлия - Просвистела.mp3",

		// === MIXED CASE (DON'T TOUCH) ===
		"Daft Punk - Get Lucky.mp3",
		"Wu-Tang - Gravel Pit.mp3",
		"Twenty One Pilots - Stressed Out.mp3",

		// === HYPHENATED NAMES ===
		"lo-fi-hip-hop-beats.mp3",
		"dark-ambient-drone.flac",
		"Wu-Tang - Gravel Pit.mp3",
		"Jay-Z - Empire State of Mind.mp3",

		// === NO SEPARATOR ===
		"some random filename.mp3",
		"just a song.flac",
		"one.mp3",

		// === FILE PATHS ===
		"/music/Artist - Title.mp3",
		"C:\\Music\\Artist - Title.mp3",
		"/home/user/downloads/01. Artist - Title (Remix).flac",

		// === GARBAGE CHARACTERS ===
		"~~Artist - Title~~.mp3",
		"~~~Artist - Title~~~.mp3",
		"###Artist - Title.mp3",
		"@@@Artist - Title@@@.mp3",
		"***Artist - Title***.mp3",
		"$$$Artist - Title$$$.mp3",
		"===Artist - Title===.mp3",
		"~~~EMINEM - RAP GOD~~~.mp3",

		// === COMPLEX REAL-WORLD GARBAGE ===
		"DJ Khaled + Drake + Lil Wayne - I'm The One (DJ Dog Remix).wav",
		"01. Avicii feat. Aloe Blacc - Wake Me Up (Remix) [320kbps].mp3",
		"#3 Skrillex + Diplo ft. Justin Bieber - Where Are U Now (Official Video).mp3",
		"NF - THE SEARCH_05.mp3",
		"drake_take_care_feat_rihanna_987654321.mp3",
		"XXXTENTACION - SAD! (Official Music Video).mp3",
		"Marshmello + Anne-Marie - FRIENDS (Music Video).mp3",
		"post_malone_rockstar_ft_21_savage_12345678_abcdef01.mp3",

		// === VARIOUS EXTENSIONS ===
		"Artist - Title.wav",
		"Artist - Title.ogg",
		"Artist - Title.m4a",
		"Artist - Title.opus",
		"Artist - Title.wma",
		"Artist - Title.aiff",
		"Artist - Title.aac",
		"Artist - Title.flac",

		// === EDGE CASES ===
		"- Artist - Title.mp3",
		"Artist - Title -.mp3",
		"  Artist  -  Title  .mp3",
		"Artist - Title (Remix) (Live) (Remastered 2011) [Explicit].mp3",
		"01.Artist-Title.mp3",
		"1-Artist-Title.mp3",

		// === BIZARRE BUT POSSIBLE ===
		"unknown_artist_unknown_title.mp3",
		"track_01.mp3",
		"audio_recording_2024_01_15.mp3",
		"voice_memo_final_v2.mp3",
		"song (1).mp3",
		"New Recording 2.mp3",
		"Запись голоса 003.mp3",

		// === MULTI-LANGUAGE ===
		"坂本龍一 - Merry Christmas Mr Lawrence.mp3",
		"BTS - 봄날.mp3",
		"Stromae - Alors on danse.mp3",
		"Rammstein - Du hast.mp3",
		"Alizée - Moi... Lolita.mp3",

		// === FEATURING VARIATIONS ===
		"Artist Feat. B - Song.mp3",
		"Artist FEAT. B - Song.mp3",
		"Artist FT. B - Song.mp3",
		"Artist Ft B - Song.mp3",
		"Artist (Feat. B) - Song.mp3",
		"Artist - Song Feat. B.mp3",
		"Artist - Song (Ft. B & C).mp3",

		// === EXTREMELY GARBAGE ===
		"~~~___BEST---MUSIC===2024___~~~.mp3",
		"~~~~.mp3",
		"12345678.mp3",
		"abcdef0123456789.mp3",
		"__________test__________.mp3",
		"---artist---title---.mp3",
		"[HQ] Artist - Title [320kbps] (Official).mp3",
		"(1) Artist - Title.mp3",
		"{Remix} Artist - Title.mp3",

		// === DOUBLE/TRIPLE SEPARATORS ===
		"Artist - - Title.mp3",
		"Artist - Title - Subtitle.mp3",
		"A - B - C - D.mp3",

		// === MIXED SEPARATORS ===
		"Artist — Title (Remix) [Explicit] {Live}.mp3",
		"01. Artist + B feat. C - Title (Remix) [320kbps].ogg",

		// === ACTUAL TRICKY FILENAMES FROM THE WILD ===
		"Linkin Park - In The End (Official Video) [HD].mp3",
		"Eminem - Lose Yourself (8 Mile Soundtrack).mp3",
		"Michael Jackson - Beat It (2008 Remaster).flac",
		"The Beatles - Let It Be (Remastered 2009).flac",
		"Nirvana - Smells Like Teen Spirit (Official Music Video).mp3",
		"Red Hot Chili Peppers - Californication (Official Music Video).mp3",
		"Imagine Dragons - Believer (Lyrics Video).mp3",
		"Ed Sheeran - Shape of You (Official Audio).mp3",
		"Billie Eilish - Bad Guy (Remix) (Free Download).mp3",
		"The Weeknd + Daft Punk - Starboy (HQ).mp3",
	}

	var sb strings.Builder
	sb.WriteString("=== AudioInk Parser Test Report ===\n")
	sb.WriteString(fmt.Sprintf("Total test cases: %d\n", len(cases)))
	sb.WriteString("Format: INPUT → Artist | Title | Extras | Featured | Track | Confidence\n")
	sb.WriteString(strings.Repeat("=", 120) + "\n\n")

	passed := 0
	issues := 0

	for i, input := range cases {
		r := Parse(input)

		line := fmt.Sprintf("%3d) %s\n", i+1, input)
		sb.WriteString(line)

		out := fmt.Sprintf("     → Artist: %q | Title: %q", r.Artist, r.Title)
		if r.Extras != "" {
			out += fmt.Sprintf(" | Extras: %q", r.Extras)
		}
		if len(r.FeaturedArtists) > 0 {
			out += fmt.Sprintf(" | Feat: %v", r.FeaturedArtists)
		}
		if r.Track > 0 {
			out += fmt.Sprintf(" | Track: %d", r.Track)
		}
		out += fmt.Sprintf(" | Confidence: %s", r.Confidence)
		sb.WriteString(out + "\n")

		// Flag potential issues
		hasIssue := false
		if r.Artist == "" && r.Title == "" {
			sb.WriteString("     ⚠ WARNING: Both artist and title are empty!\n")
			hasIssue = true
		}
		if r.Title == "" {
			sb.WriteString("     ⚠ WARNING: Title is empty!\n")
			hasIssue = true
		}
		if strings.ContainsAny(r.Artist, "~#@=%^*`|\\;") {
			sb.WriteString("     ⚠ WARNING: Artist contains garbage chars!\n")
			hasIssue = true
		}
		if strings.ContainsAny(r.Title, "~#@=%^*`|\\;") {
			sb.WriteString("     ⚠ WARNING: Title contains garbage chars!\n")
			hasIssue = true
		}
		if strings.Contains(r.Artist, "  ") || strings.Contains(r.Title, "  ") {
			sb.WriteString("     ⚠ WARNING: Double spaces detected!\n")
			hasIssue = true
		}
		// Check for leftover garbage IDs in artist/title
		words := strings.Fields(r.Artist + " " + r.Title)
		for _, w := range words {
			if len(w) >= 6 && isGarbageId(w) {
				sb.WriteString(fmt.Sprintf("     ⚠ WARNING: Garbage ID %q still in output!\n", w))
				hasIssue = true
				break
			}
		}

		if hasIssue {
			issues++
		} else {
			passed++
		}

		sb.WriteString("\n")
	}

	sb.WriteString(strings.Repeat("=", 120) + "\n")
	sb.WriteString(fmt.Sprintf("RESULTS: %d/%d clean (%d flagged for review)\n", passed, len(cases), issues))

	// Write to file
	err := os.WriteFile("../donetests.txt", []byte(sb.String()), 0644)
	if err != nil {
		t.Fatalf("Failed to write donetests.txt: %v", err)
	}

	t.Logf("Report written to donetests.txt (%d cases, %d clean, %d flagged)", len(cases), passed, issues)
}
