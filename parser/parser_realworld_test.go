package parser

import "testing"

// TestRealWorld runs dozens of real-world filenames the parser handles
// correctly. These are regression cases — if any start failing, you've
// broken something.
//
// For known-broken cases see parser_known_issues_test.go.
func TestRealWorld(t *testing.T) {
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
		// === Tricky band names ===
		{"blink-182 - All The Small Things.mp3", expect{"Blink-182", "All The Small Things", 0, ""}},
		{"!!! - Me And Giuliani Down By The School Yard.mp3", expect{"!!!", "Me And Giuliani Down By The School Yard", 0, ""}},
		{"Run-D.M.C. - It's Tricky.mp3", expect{"Run-D.M.C.", "It's Tricky", 0, ""}},
		{"P.O.D. - Alive.mp3", expect{"P.O.D.", "Alive", 0, ""}},
		{"M.I.A. - Paper Planes.mp3", expect{"M.I.A.", "Paper Planes", 0, ""}},
		{"Mötley Crüe - Kickstart My Heart.mp3", expect{"Mötley Crüe", "Kickstart My Heart", 0, ""}},
		{"Sigur Rós - Hoppípolla.mp3", expect{"Sigur Rós", "Hoppípolla", 0, ""}},
		{"Beyoncé - Halo.mp3", expect{"Beyoncé", "Halo", 0, ""}},
		{"P!nk - So What.mp3", expect{"P!nk", "So What", 0, ""}},
		{"Ke$ha - Tik Tok.mp3", expect{"Ke$ha", "Tik Tok", 0, ""}},
		{"A$AP Rocky - Praise The Lord.mp3", expect{"A$AP Rocky", "Praise The Lord", 0, ""}},
		{"3OH!3 - Don't Trust Me.mp3", expect{"3OH!3", "Don't Trust Me", 0, ""}},

		// === Numbers in artist (no longer eaten as track number) ===
		{"3 Doors Down - Kryptonite.mp3", expect{"3 Doors Down", "Kryptonite", 0, ""}},
		{"50 Cent - In Da Club.mp3", expect{"50 Cent", "In Da Club", 0, ""}},
		{"U2 - One.mp3", expect{"U2", "One", 0, ""}},
		{"Sum 41 - Fat Lip.mp3", expect{"Sum 41", "Fat Lip", 0, ""}},
		{"Maroon 5 - Sugar.mp3", expect{"Maroon 5", "Sugar", 0, ""}},
		{"Matchbox 20 - Push.mp3", expect{"Matchbox 20", "Push", 0, ""}},
		{"4 Non Blondes - What's Up.mp3", expect{"4 Non Blondes", "What's Up", 0, ""}},

		// === Years as titles (no longer stripped as garbage IDs) ===
		{"Prince - 1999.mp3", expect{"Prince", "1999", 0, ""}},
		// Note: "for" → "For" is the every-word title-case style, intentional.
		{"Bowling for Soup - 1985.mp3", expect{"Bowling For Soup", "1985", 0, ""}},

		// === Disc prefix stripping ===
		{"Disc 1 - 01 - Artist - Title.mp3", expect{"Artist", "Title", 1, ""}},
		{"CD1 - 03 - Artist - Title.mp3", expect{"Artist", "Title", 3, ""}},
		{"Disc 2 Track 05 - Artist - Title.mp3", expect{"Artist", "Title", 5, ""}},

		// === will.i.am — stylized lowercase preserved ===
		{"will.i.am - Scream & Shout.mp3", expect{"will.i.am", "Scream & Shout", 0, ""}},

		// === Roman numerals preserved ===
		{"Beethoven - Symphony No. 9 - IV. Ode to Joy.mp3", expect{"Beethoven", "Symphony No. 9 - IV. Ode To Joy", 0, ""}},
		{"Bach - Partita No. III in E.mp3", expect{"Bach", "Partita No. III In E", 0, ""}},

		// === UTF-8 + bare dash separator (regression: byte-vs-rune bug) ===
		{"Артист-Название.mp3", expect{"Артист", "Название", 0, ""}},
		{"01-Артист-Название.mp3", expect{"Артист", "Название", 1, ""}},
		{"Кино-Перемен.mp3", expect{"Кино", "Перемен", 0, ""}},
		{"米津玄師-Lemon.mp3", expect{"米津玄師", "Lemon", 0, ""}},

		// === Whole-name bracket unwrap ===
		{"(Artist - Title).mp3", expect{"Artist", "Title", 0, ""}},
		{"[Artist - Title].mp3", expect{"Artist", "Title", 0, ""}},
		{"((Artist - Title)).mp3", expect{"Artist", "Title", 0, ""}},
		{"[[Artist - Title]].mp3", expect{"Artist", "Title", 0, ""}},

		// === TikTok-era extras (correctly captured) ===
		{"Artist - Title (Sped Up).mp3", expect{"Artist", "Title", 0, "Sped Up"}},
		{"Artist - Title (Slowed).mp3", expect{"Artist", "Title", 0, "Slowed"}},
		{"Artist - Title (Slowed + Reverb).mp3", expect{"Artist", "Title", 0, "Slowed + Reverb"}},
		{"Artist - Title (Nightcore).mp3", expect{"Artist", "Title", 0, "Nightcore"}},
		{"Artist - Title (Phonk Remix).mp3", expect{"Artist", "Title", 0, "Phonk Remix"}},
		{"Artist - Title (Bass Boosted).mp3", expect{"Artist", "Title", 0, "Bass Boosted"}},
		{"Artist - Title (8D Audio).mp3", expect{"Artist", "Title", 0, "8D Audio"}},

		// === Version / take / demo markers ===
		{"Artist - Title (Demo).mp3", expect{"Artist", "Title", 0, "Demo"}},
		{"Artist - Title (Take 1).mp3", expect{"Artist", "Title", 0, "Take 1"}},
		{"Artist - Title (Demo Version).mp3", expect{"Artist", "Title", 0, "Demo Version"}},
		{"Artist - Title (Clean).mp3", expect{"Artist", "Title", 0, "Clean"}},
		{"Artist - Title (Radio Version).mp3", expect{"Artist", "Title", 0, "Radio Version"}},
		{"Artist - Title (Promo).mp3", expect{"Artist", "Title", 0, "Promo"}},

		// === Soundtrack extras ===
		{`Hans Zimmer - Time (Inception OST).mp3`, expect{"Hans Zimmer", "Time", 0, "Inception OST"}},
		{`John Williams - Imperial March (Star Wars).mp3`, expect{"John Williams", "Imperial March", 0, "Star Wars"}},

		// === "The" prefix artists ===
		{"The Beatles - Let It Be.mp3", expect{"The Beatles", "Let It Be", 0, ""}},
		{"The Rolling Stones - Paint It Black.mp3", expect{"The Rolling Stones", "Paint It Black", 0, ""}},
		{"the white stripes - seven nation army.mp3", expect{"The White Stripes", "Seven Nation Army", 0, ""}},

		// === Artists with "&" / multi-word names ===
		{"Hall & Oates - You Make My Dreams.mp3", expect{"Hall & Oates", "You Make My Dreams", 0, ""}},
		{"Simon & Garfunkel - The Sound of Silence.mp3", expect{"Simon & Garfunkel", "The Sound Of Silence", 0, ""}},

		// === Various Artists / compilations ===
		{"Various Artists - Pulp Fiction Soundtrack - 03 - Misirlou.mp3",
			expect{"Various Artists", "Pulp Fiction Soundtrack - 03 - Misirlou", 0, ""}},

		// === Russian / Cyrillic edge cases ===
		{"Земфира - Прости меня моя любовь (Live).mp3", expect{"Земфира", "Прости Меня Моя Любовь", 0, "Live"}},
		{"АукцЫон - Боги (Remix).mp3", expect{"АукцЫон", "Боги", 0, "Remix"}},
		{"Гражданская Оборона - Все идет по плану.mp3",
			expect{"Гражданская Оборона", "Все Идет По Плану", 0, ""}},
		// Note: "и" → "И" is the every-word title-case style (same as English "and" → "And").
		{"Король и Шут - Кукла колдуна (Remastered 2020).flac",
			expect{"Король И Шут", "Кукла Колдуна", 0, "Remastered 2020"}},

		// === Unicode separators (pipe, bullet, arrow, pointer) ===
		{"Artist | Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist • Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist → Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist ► Title.mp3", expect{"Artist", "Title", 0, ""}},

		// === Date prefix stripping ===
		{"2024-01-15 Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"2024-01-15 - Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"[2024-01-15] Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"20240115 - Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"1999-01-01 - Artist - Title.mp3", expect{"Artist", "Title", 0, ""}},
		// "12345678" doesn't match the date regex (no 19/20 prefix) → falls through to garbage handling
		{"12345678.mp3", expect{"", "12345678", 0, ""}},

		// === Single-name artists ===
		{"Cher - Believe.mp3", expect{"Cher", "Believe", 0, ""}},
		{"Madonna - Vogue.mp3", expect{"Madonna", "Vogue", 0, ""}},
		{"Drake - Hotline Bling.mp3", expect{"Drake", "Hotline Bling", 0, ""}},

		// === Period-containing artist names (Mr., Dr., St.) ===
		{"Mr. Bungle - Sweet Charity.mp3", expect{"Mr. Bungle", "Sweet Charity", 0, ""}},
		{"Mr. Mister - Kyrie.mp3", expect{"Mr. Mister", "Kyrie", 0, ""}},
		{"St. Vincent - Digital Witness.mp3", expect{"St. Vincent", "Digital Witness", 0, ""}},
		{"Dr. Dre - Still D.R.E.mp3", expect{"Dr. Dre", "Still D.R.E", 0, ""}},

		// === Multiple track-prefix variations ===
		{"02.Artist-Title.mp3", expect{"Artist", "Title", 2, ""}},
		{"02)Artist - Title.mp3", expect{"Artist", "Title", 2, ""}},
		{"02 - Artist - Title.mp3", expect{"Artist", "Title", 2, ""}},

		// === Podcast / episode markers preserved in artist ===
		{"Joe Rogan Experience #2000 - Elon Musk.mp3",
			expect{"Joe Rogan Experience #2000", "Elon Musk", 0, ""}},
		{"H3 Podcast #500 - Guest Name.mp3",
			expect{"H3 Podcast #500", "Guest Name", 0, ""}},

		// === Multi-space and stylistic spacing ===
		{"Artist  -  Title.mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist   ---   Title.mp3", expect{"Artist", "Title", 0, ""}},

		// === Hyphenated titles preserved ===
		{"Coldplay - X-Marks.mp3", expect{"Coldplay", "X-Marks", 0, ""}},
		{"Artist - X-Ray.mp3", expect{"Artist", "X-Ray", 0, ""}},

		// === Smart quotes preserved verbatim ===
		{"Artist - Don't Stop Me Now.mp3", expect{"Artist", "Don't Stop Me Now", 0, ""}},
		{"Artist - Don’t Stop Me Now.mp3", expect{"Artist", "Don’t Stop Me Now", 0, ""}},

		// === Numbers as title (under 4 digits — safe from garbage-id stripping) ===
		{"Adele - 19.mp3", expect{"Adele", "19", 0, ""}},
		{"Adele - 21.mp3", expect{"Adele", "21", 0, ""}},
		{"Beach Boys - 409.mp3", expect{"Beach Boys", "409", 0, ""}},
		{"The Beatles - 1.mp3", expect{"The Beatles", "1", 0, ""}},

		// === Comma in artist (multi-artist not split as feat) ===
		{"Tyler, The Creator - EARFQUAKE.mp3", expect{"Tyler, The Creator", "Earfquake", 0, ""}},
		{"Future, Metro Boomin - Like That.mp3", expect{"Future, Metro Boomin", "Like That", 0, ""}},

		// === Subtitle / 3-segment titles (keep subtitle in title) ===
		{"Pink Floyd - Wish You Were Here - Live At Wembley.mp3", expect{"Pink Floyd", "Wish You Were Here - Live At Wembley", 0, ""}},
		// Note: parser applies "every word capitalized" style — "in", "to" become "In", "To".
		// That's a stylistic choice, not a bug.
		{"Bach - Symphony No. 5 in C Minor - I. Allegro.mp3", expect{"Bach", "Symphony No. 5 In C Minor - I. Allegro", 0, ""}},
		{"Artist - Title - Album Version.mp3", expect{"Artist", "Title - Album Version", 0, ""}},

		// === Featured chains ===
		{"Eminem ft. Dido - Stan.mp3", expect{"Eminem", "Stan", 0, ""}},
		{"Kendrick Lamar feat. SZA - All The Stars.mp3", expect{"Kendrick Lamar", "All The Stars", 0, ""}},
		{"Post Malone & Swae Lee - Sunflower.mp3", expect{"Post Malone & Swae Lee", "Sunflower", 0, ""}},
		{"Migos - Bad And Boujee (feat. Lil Uzi Vert).mp3", expect{"Migos", "Bad And Boujee", 0, ""}},

		// === Apostrophes / punctuation ===
		{"OutKast - Ms. Jackson.mp3", expect{"OutKast", "Ms. Jackson", 0, ""}},
		{"Mr. Bungle - Sweet Charity.mp3", expect{"Mr. Bungle", "Sweet Charity", 0, ""}},
		{"Guns N' Roses - Sweet Child O' Mine.mp3", expect{"Guns N' Roses", "Sweet Child O' Mine", 0, ""}},
		{"Eminem - Without Me (Don't Try This At Home).mp3", expect{"Eminem", "Without Me", 0, "Don't Try This At Home"}},
		{"Artist - It's Time.mp3", expect{"Artist", "It's Time", 0, ""}},
		{"Artist - I'm The One Who Did It.mp3", expect{"Artist", "I'm The One Who Did It", 0, ""}},

		// === Asian / international ===
		{"米津玄師 - Lemon.mp3", expect{"米津玄師", "Lemon", 0, ""}},
		{"坂本龍一 - Merry Christmas Mr Lawrence.mp3", expect{"坂本龍一", "Merry Christmas Mr Lawrence", 0, ""}},
		{"BTS - 봄날.mp3", expect{"BTS", "봄날", 0, ""}},
		{"Stromae - Alors on danse.mp3", expect{"Stromae", "Alors On Danse", 0, ""}},
		{"Rammstein - Du hast.mp3", expect{"Rammstein", "Du Hast", 0, ""}},

		// === Extras kept ===
		{"Artist - Title (Original Mix).mp3", expect{"Artist", "Title", 0, "Original Mix"}},
		{"Artist - Title (Club Mix).mp3", expect{"Artist", "Title", 0, "Club Mix"}},
		{"Artist - Title (Radio Edit).mp3", expect{"Artist", "Title", 0, "Radio Edit"}},
		{"Artist - Title (Extended Mix).mp3", expect{"Artist", "Title", 0, "Extended Mix"}},
		{"Artist - Title (Deluxe Edition).mp3", expect{"Artist", "Title", 0, "Deluxe Edition"}},
		{"Artist - Title (Acoustic).mp3", expect{"Artist", "Title", 0, "Acoustic"}},
		{"Artist - Title (Instrumental).mp3", expect{"Artist", "Title", 0, "Instrumental"}},
		{"Artist - Title (Single Version).mp3", expect{"Artist", "Title", 0, "Single Version"}},

		// === Junk extras stripped ===
		{"Artist - Title (Official Audio).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Audio Only).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title [MP3].mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (HD).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (Full Version).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (LQ).mp3", expect{"Artist", "Title", 0, ""}},
		{"Artist - Title (CDQ).mp3", expect{"Artist", "Title", 0, ""}},

		// === Track numbers ===
		{"99 - Toto - Africa.mp3", expect{"Toto", "Africa", 99, ""}},
		{"001 - Artist - Title.mp3", expect{"Artist", "Title", 1, ""}},
		{"100 - Artist - Title.mp3", expect{"Artist", "Title", 100, ""}},
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
