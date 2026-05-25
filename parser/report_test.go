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

		// === REAL BAND NAMES WITH TRICKY CHARS ===
		"AC/DC - Highway to Hell.mp3",
		"blink-182 - All The Small Things.mp3",
		"!!! - Me And Giuliani Down By The School Yard.mp3",
		"$uicideboy$ - Kill Yourself.mp3",
		"Mötley Crüe - Kickstart My Heart.mp3",
		"Bjørn Solo - Album Track.mp3",
		"Sigur Rós - Hoppípolla.mp3",
		"Beyoncé - Halo.mp3",
		"Mø - Final Song.mp3",
		"P!nk - So What.mp3",
		"Ke$ha - Tik Tok.mp3",
		"A$AP Rocky - Praise The Lord.mp3",
		"DJ E-Z Rock - It Takes Two.mp3",
		"will.i.am - Scream & Shout.mp3",
		"M.I.A. - Paper Planes.mp3",
		"Run-D.M.C. - It's Tricky.mp3",
		"P.O.D. - Alive.mp3",
		"+44 - When Your Heart Stops Beating.mp3",
		"3 Doors Down - Kryptonite.mp3",
		"50 Cent - In Da Club.mp3",
		"24kGoldn - Mood.mp3",
		"U2 - One.mp3",
		"Sum 41 - Fat Lip.mp3",
		"Maroon 5 - Sugar.mp3",
		"Matchbox 20 - Push.mp3",

		// === NUMBERS IN TITLES (NOT TRACK NUMBERS!) ===
		"Bowling for Soup - 1985.mp3",
		"Prince - 1999.mp3",
		"Adele - 19.mp3",
		"Adele - 21.mp3",
		"Beach Boys - 409.mp3",
		"Drake - 0 To 100.mp3",
		"The Beatles - 1.mp3",

		// === HIP-HOP COMPLEX FEAT CHAINS ===
		"DJ Khaled feat. Drake, Lil Wayne, Big Sean - Lavish.mp3",
		"Eminem ft. Dido - Stan.mp3",
		"Kendrick Lamar feat. SZA - All The Stars.mp3",
		"Post Malone & Swae Lee - Sunflower.mp3",
		"21 Savage x Metro Boomin - Mr. Right Now.mp3",
		"Future, Metro Boomin - Like That.mp3",
		"Migos - Bad And Boujee (feat. Lil Uzi Vert).mp3",
		"Tyler, The Creator - EARFQUAKE.mp3",

		// === FEATURING VARIANT KEYWORDS ===
		"Artist x B - Song.mp3",
		"Artist X B - Song.mp3",
		"Artist X B X C - Song.mp3",
		"Artist with B - Song.mp3",
		"Artist presents B - Song.mp3",
		"Artist pres. B - Song.mp3",
		"Artist vs Artist2 - Battle.mp3",
		"Artist VS Artist2 - Battle.mp3",
		"Artist VS. Artist2 - Battle.mp3",
		"Artist + B & C - Song.mp3",

		// === SUBTITLE / MULTIPLE DASHES IN TITLE ===
		"Pink Floyd - Wish You Were Here - Live At Wembley.mp3",
		"Bach - Symphony No. 5 in C Minor - I. Allegro.mp3",
		"Mozart - Sonata K. 545 - II. Andante.mp3",
		"Artist - Title - Album Version.mp3",
		"Artist - Title - Single Edit.mp3",
		"Beethoven - Symphony No. 9 - IV. Ode to Joy.mp3",

		// === COMPILATION / DISC / TRACK FORMATS ===
		"VA - Best of 2023 - 01 - Artist - Song.mp3",
		"Various Artists - Now Vol. 50 - 03 - Artist - Title.mp3",
		"Disc 1 - 01 - Artist - Title.mp3",
		"CD1 - 03 - Artist - Title.mp3",
		"Disc 2 Track 05 - Artist - Title.mp3",
		"1-01 Artist - Title.mp3",
		"1.01 Artist - Title.mp3",
		"The Beatles - 1.01 - Love Me Do.mp3",

		// === STREAMING PLATFORM CONVENTIONS ===
		"y2mate.com - Artist - Title.mp3",
		"[Spotify] Artist - Title.mp3",
		"Artist - Title (Original Mix).mp3",
		"Artist - Title (Club Mix).mp3",
		"Artist - Title (Dub Mix).mp3",
		"Artist - Title (Edit).mp3",
		"Artist - Title (Single Version).mp3",
		"Artist - Title - YouTube.mp3",

		// === BRACKETS AT START ===
		"(Bonus) Artist - Title.mp3",
		"[Live] Artist - Title.mp3",
		"[2023] Artist - Album - 01 Song.mp3",
		"[HQ] Artist - Title.mp3",
		"[Acoustic] Artist - Title.mp3",

		// === APOSTROPHES / PUNCTUATION ===
		"Guns N' Roses - Sweet Child O' Mine.mp3",
		"Don't Stop Believin' - Journey.mp3",
		"OutKast - Ms. Jackson.mp3",
		"Mr. Bungle - Sweet Charity.mp3",
		"Dr. Dre - Still D.R.E.mp3",
		"Ms. Lauryn Hill - Doo Wop.mp3",
		"Eminem - Without Me (Don't Try This At Home).mp3",
		"Artist - It's Time.mp3",
		"Artist - I'm The One Who Did It.mp3",

		// === MIXED LANGUAGE / EMOJI / SPECIAL UNICODE ===
		"BLACKPINK - 뚜두뚜두 (DDU-DU DDU-DU).mp3",
		"NewJeans - OMG.mp3",
		"米津玄師 - Lemon.mp3",
		"YOASOBI - Idol.mp3",
		"Drake - 🍑.mp3",
		"Artist - Song 🔥.mp3",
		"♥ Artist - Title ♥.mp3",
		"Artist — Title (★ Remix ★).mp3",

		// === REAL CLASSICAL NAMING ===
		"Tchaikovsky - 1812 Overture, Op. 49.mp3",
		"Beethoven - Piano Sonata No. 14 in C-sharp Minor, Op. 27 No. 2 'Moonlight'.mp3",
		"Bach, J.S. - Toccata and Fugue in D Minor, BWV 565.mp3",
		"Mozart - Requiem in D Minor, K. 626 - Lacrimosa.mp3",

		// === EDGE: TRACK NUMBER VS YEAR VS BARE NUMBER ===
		"1985 - Bowling for Soup.mp3",       // ambiguous: track 1985? or year?
		"2001 - A Space Odyssey.mp3",
		"99 - Toto - Africa.mp3",            // track 99
		"100 - Artist - Title.mp3",          // 3-digit track
		"01-02 - Artist - Title.mp3",        // disc-track format
		"Artist - Track 03.mp3",             // bizarre track in title

		// === BAD UTF-8 / WEIRD WHITESPACE ===
		"Artist - Title.mp3",       // non-breaking space (U+00A0)
		"Artist - Title.mp3",       // em space (U+2003)
		"Artist​-Title.mp3",         // zero-width space (invisible!)
		"Artist - Title​.mp3",       // trailing ZWSP in title

		// === DOUBLE EXTENSIONS / WEIRD EXT CASING ===
		"Artist - Title.mp3.mp3",   // accidental double extension
		"Artist - Title.MP3",       // uppercase ext
		"Artist - Title.Mp3",       // mixed case ext
		"Artist - Title.flac.flac",

		// === EXTREMELY LONG ===
		"Artist - This Is A Really Really Really Really Really Really Long Title That Should Still Parse Fine Even If It Has Many Words.mp3",

		// === ONE-CHAR NAMES ===
		"X - Y.mp3",
		"A - B.mp3",
		"1 - 2.mp3",

		// === PARENTHESES IN ARTIST ===
		"Artist (Solo Project) - Title.mp3",
		"The Band (UK) - Song.mp3",
		"Twisted Sister (Live in NYC) - We're Not Gonna Take It.mp3",

		// === KNOWN PARSER GOTCHAS (Picard/beets reported bugs) ===
		"Artist AND B - Song.mp3",            // Picard crashes on uppercase AND
		"Artist OR B - Song.mp3",
		"UB40 - Red Red Wine.mp3",            // UB40 mistaken as track number by Picard
		"S Club 7 - S Club Party.mp3",        // number-suffix band name
		"Tha Eastsidaz feat. Snoop Dogg - G'd Up.mp3",
		"4 Non Blondes - What's Up.mp3",
		"3OH!3 - Don't Trust Me.mp3",

		// === STREAMING / VIRAL CONVENTIONS (TikTok era) ===
		"Artist - Title (Sped Up).mp3",
		"Artist - Title (Slowed).mp3",
		"Artist - Title (Slowed + Reverb).mp3",
		"Artist - Title (Slowed Down).mp3",
		"Artist - Title (Nightcore).mp3",
		"Artist - Title (Tiktok Version).mp3",
		"Artist - Title (TikTok Remix).mp3",
		"Artist - Title (Phonk Remix).mp3",
		"Artist - Title (Drift Phonk).mp3",
		"Artist - Title (8D Audio).mp3",
		"Artist - Title (Bass Boosted).mp3",
		"Artist - Title (Earrape Edition).mp3",

		// === SOUNDTRACK / OST CONVENTIONS ===
		`Artist - Title (From "Movie Title").mp3`,
		"Artist - Title [From The Motion Picture].mp3",
		"Artist - Title (OST).mp3",
		"Artist - Title - OST Inception.mp3",
		"Hans Zimmer - Time (Inception OST).mp3",
		"John Williams - Imperial March (Star Wars).mp3",
		"Trent Reznor & Atticus Ross - The Social Network (OST) - 01 - Hand Covers Bruise.mp3",
		"Various Artists - Pulp Fiction Soundtrack - 03 - Misirlou.mp3",

		// === MULTI-DISC / DOUBLE ALBUM ===
		"Artist - Album (Disc 2) - 03 - Title.mp3",
		"Artist - Album [Disc 2] - 03 - Title.mp3",
		"Artist - Album - CD2 - 03 - Title.mp3",
		"Pink Floyd - The Wall - Disc 1 - Track 5 - Mother.mp3",
		"D1 - 01 - Artist - Title.mp3",
		"D2 - 11 - Artist - Title.mp3",

		// === VINYL / CASSETTE NOTATION ===
		"Artist - Title (Side A).mp3",
		"Artist - Title (Side B).mp3",
		"Artist - Title (A1).mp3",
		"Artist - Title (B3).mp3",
		"Side A - 01 - Artist - Title.mp3",
		"Artist - Album - Side One - Track 4.mp3",

		// === VERSION / TAKE / DEMO MARKERS ===
		"Artist - Title (Demo).mp3",
		"Artist - Title (Demo Version).mp3",
		"Artist - Title (Take 1).mp3",
		"Artist - Title (Take 47).mp3",
		"Artist - Title (Alt. Version).mp3",
		"Artist - Title (Alternate Take).mp3",
		"Artist - Title v2.mp3",
		"Artist - Title (v3).mp3",
		"Artist - Title (Rough Mix).mp3",
		"Artist - Title (Final Mix).mp3",
		"Artist - Title (Promo).mp3",
		"Artist - Title (Promo Single).mp3",
		"Artist - Title (Test Pressing).mp3",

		// === PART / CHAPTER / EPISODE ===
		"Artist - Title Pt. 1.mp3",
		"Artist - Title Pt. 2.mp3",
		"Artist - Title (Part 1).mp3",
		"Artist - Title - Part 2.mp3",
		"Artist - Title (Chapter 3).mp3",

		// === CLEAN / EXPLICIT / RADIO MARKERS ===
		"Artist - Title (Clean).mp3",
		"Artist - Title (Dirty).mp3",
		"Artist - Title (Radio Version).mp3",
		"Artist - Title (Censored).mp3",
		"Artist - Title (Uncensored).mp3",
		"Artist - Title (Explicit).mp3",

		// === YEAR PREFIXES / GENRE PREFIXES ===
		"(2023) Artist - Title.mp3",
		"[2024] Artist - Title.mp3",
		"[Hip-Hop] Artist - Title.mp3",
		"[Rock] Artist - Title.mp3",
		"[EDM] Artist - Title.mp3",
		"[Dnb] Artist - Title.mp3",
		"[Drum & Bass] Artist - Title.mp3",
		"[Phonk] Artist - Title.mp3",

		// === RUSSIAN / CIS TORRENT DUMP CONVENTIONS ===
		"01-Артист-Название.mp3",
		"Артист-Название (2023) [320 kbps]-01.mp3",
		"[FLAC] Артист - Альбом - 01 - Название.flac",
		"Артист - Альбом 2023 (Lossless).flac",
		"01-artist-title-(remix)-(2023).mp3",
		"Артист - Название [www.audio.ru].mp3",
		"Артист - Название (Промо).mp3",
		"Артист - Название (Сингл).mp3",
		"Артист - Название (Минусовка).mp3",
		"Артист - Название (Караоке).mp3",
		"Артист - Название (Авторский трек).mp3",

		// === DJ MIX SET TRACKS ===
		"01 - Tiësto - In Search of Sunrise - 01 - Track Name.mp3",
		"Tiësto @ Tomorrowland 2023 - 01 - Track.mp3",
		"DJ Snake - Live at Coachella 2023 - 03 - Track.mp3",
		"Above & Beyond - Group Therapy 500 - 01 - Track.mp3",

		// === DJ TAG / PRODUCER TAG NOISE ===
		"(DJ Snake Edit) Artist - Title.mp3",
		"(Prod. by Metro Boomin) Artist - Title.mp3",
		"Artist - Title (Prod. Murda Beatz).mp3",
		"Artist - Title [Prod. Tay Keith].mp3",
		"Artist - Title (Beat by Dr. Dre).mp3",

		// === BEAT TAPE / TYPE BEAT CONVENTIONS ===
		"[FREE] Drake Type Beat 2023 - 'Vibes'.mp3",
		"Drake Type Beat - 'Vibes' (Prod. ProducerName).mp3",
		"FREE Travis Scott Type Beat - 'Astro'.mp3",
		"$AVE - SOLD - Drake Type Beat.mp3",

		// === ARTIST NAME WITH NUMBERS / SYMBOLS REVISITED ===
		"+44 - When Your Heart Stops Beating.mp3",
		"!!! - Heart of Hearts.mp3",
		"???? - Mystery Track.mp3",
		"deadmau5 - Strobe.mp3",
		"deadmau5 - I Remember.mp3",
		"Skrillex - Bangarang feat. Sirah.mp3",
		"Skrillex - Scary Monsters and Nice Sprites.mp3",
		"$NOT - Gosha.mp3",
		"Yung Lean - Yoshi City.mp3",
		"Yung Gravy - Mr. Clean.mp3",
		"j-hope - on the street.mp3",
		"i-Ro - Title.mp3",

		// === MIXTAPE PREFIXES ===
		"[Mixtape] Artist - Title.mp3",
		"[MIXTAPE 2023] Artist - Title.mp3",
		"DJ Drama presents Lil Wayne - Dedication 5 - 03 - Title.mp3",

		// === EDGE: WEIRD MIDDLE / TRAILING JUNK ===
		"Artist - Title - 192kbps - downloaded from somewhere.mp3",
		"Artist - Title - high quality - 320kbps.mp3",
		"Artist - Title (no copyright music).mp3",
		"Artist - Title [NoCopyrightSounds].mp3",
		"Artist - Title [NCS Release].mp3",
		"Artist - Title (Royalty Free).mp3",
		"Artist - Title - YouTube to MP3 Converter.mp3",
		"Artist - Title (Mp3-Tag-Removed).mp3",
		"Artist - Title (track from album 'Album Name').mp3",
		"Artist - Title (with Artist2).mp3",

		// === BROKEN / NEARLY-BROKEN INPUTS ===
		"Artist -.mp3",
		"- Title.mp3",
		"Artist -- Title.mp3",
		" - .mp3",
		".mp3",
		"Artist (Title).mp3",
		"Artist [Title].mp3",
		"(Artist - Title).mp3",
		"[Artist - Title].mp3",
		"((Artist)) - ((Title)).mp3",

		// === CYRILLIC EDGE CASES ===
		"01. Виктор Цой - Группа крови.mp3",
		"Цой - Перемен (feat. Кино).mp3",
		"Земфира - Прости меня моя любовь (Live).mp3",
		"Сплин - Романс (Acoustic).mp3",
		"АукцЫон - Боги (Remix).mp3",
		"Гражданская Оборона - Все идет по плану.mp3",
		"Король и Шут - Кукла колдуна (Remastered 2020).flac",

		// === ARTISTS WITH "THE" PREFIX ===
		"The Beatles - Let It Be.mp3",
		"The Rolling Stones - Paint It Black.mp3",
		"The Doors - Light My Fire.mp3",
		"the white stripes - seven nation army.mp3",
		"the strokes - last nite.mp3",

		// === ARTISTS WITH "&" / "and" IN NAME ===
		"Hall & Oates - You Make My Dreams.mp3",
		"Simon & Garfunkel - The Sound of Silence.mp3",
		"Sam & Dave - Soul Man.mp3",
		"Earth, Wind & Fire - September.mp3",
		"Above & Beyond - Sun and Moon.mp3",
		"Iron & Wine - Such Great Heights.mp3",
		"Of Monsters and Men - Little Talks.mp3",
		"Florence and the Machine - Dog Days Are Over.mp3",

		// === COMPOUND TITLE WITH PUNCTUATION ===
		"Artist - I Don't Want to Set The World on Fire.mp3",
		"Artist - Eleanor Rigby (Live at Shea Stadium).mp3",
		"Artist - Bohemian Rhapsody - Mama, just killed a man.mp3",

		// === HYPHEN-CONTAINING TITLES ===
		"Coldplay - X-Marks.mp3",
		"Eminem - 'Till I Collapse.mp3",
		"Eminem - 8 Mile.mp3",
		"Artist - X-Ray.mp3",
		"Artist - In-Between.mp3",
		"Artist - A-Side.mp3",

		// === FILE NAMING EDGE: NO ARTIST NAME, JUST DATE ===
		"2024-01-15 - Recording.mp3",
		"2024_01_15_recording.mp3",
		"recording_2024-01-15.mp3",
		"voice_memo_2024-12-31_23-59-59.mp3",

		// === ALBUM IN FILENAME ===
		"Artist - Album - 01 - Title.mp3",
		"Artist - [Album] - 01 - Title.mp3",
		"Artist - Title [Album].mp3",

		// === SMART QUOTES vs STRAIGHT QUOTES (Unicode subtleties) ===
		"Artist - Don't Stop Me Now.mp3",        // straight apostrophe (U+0027)
		"Artist - Don’t Stop Me Now.mp3",   // curly apostrophe (U+2019)
		"Artist - Donʼt Stop Me Now.mp3",   // modifier letter apostrophe (U+02BC)
		"Artist - “Title”.mp3",        // curly double quotes
		"Artist - \"Title\".mp3",                 // straight double quotes
		"Artist - 'Title'.mp3",                   // straight single quotes
		"Artist - «Title».mp3",        // French guillemets « »
		"Artist - „Title“.mp3",        // German low-9 quotation

		// === UNICODE SEPARATORS ===
		"Artist • Title.mp3",      // bullet point •
		"Artist ► Title.mp3",      // right-pointing pointer ►
		"Artist → Title.mp3",      // rightwards arrow →
		"Artist | Title.mp3",            // pipe (currently a garbage char)
		"Artist \\ Title.mp3",           // backslash (Windows path separator)
		"Artist / Title.mp3",            // forward slash (gets eaten by filepath.Base)
		"Artist :: Title.mp3",           // double colon
		"Artist <> Title.mp3",           // angle brackets

		// === YEAR-ALBUM-TRACK SCHEMES ===
		"Artist - 1969 - Album Title - 01 - Song.mp3",
		"Artist - [1969] Album Title - 01 - Song.mp3",
		"Artist - (1969) Album Title - 01 - Song.mp3",
		"01 - The Beatles - 1969 - Abbey Road - Come Together.mp3",
		"The Beatles - Abbey Road [1969] - 01 - Come Together.mp3",

		// === BANDCAMP / SOUNDCLOUD DOWNLOAD CONVENTIONS ===
		"Artist - 01 Title.mp3",          // Bandcamp default (no dash between track and title)
		"Artist - 02. Title.mp3",         // Bandcamp with dot
		"Artist - Title (free download).mp3",  // SoundCloud
		"Artist - Title [Free Download].mp3",
		"Artist - Title - SoundCloud.mp3",
		"Artist - Title - downloaded with JDownloader.mp3",
		"YouTube to MP3 - Artist - Title.mp3",

		// === MUSICBRAINZ PICARD DEFAULTS ===
		"Artist - Album - 01 - Title.mp3",         // matches Picard default
		"01-01 Title.mp3",                          // Picard with disc-track
		"01.01 Title.mp3",                          // Picard alt
		"Artist - Album/01 Title.mp3",              // Picard hierarchical (forward slash!)

		// === BOOTLEG / LIVE CONVENTIONS ===
		"Artist - Title (Live Bootleg 1985 Berlin).mp3",
		"Artist - Title (Live at Madison Square Garden, 1985).mp3",
		"Artist - Title [Bootleg].mp3",
		"Artist - Title (Studio Bootleg).mp3",
		"Artist - Title (Soundboard).mp3",
		"Artist - Title (FM Broadcast 1985).mp3",
		"Artist - Title (Radio Broadcast).mp3",
		"Bruce Springsteen - Born to Run (Live - Hammersmith Odeon, London - 1975).mp3",

		// === LENGTH MARKERS (DJ/Vinyl) ===
		"Artist - Title (12'' Mix).mp3",
		"Artist - Title (7\" Edit).mp3",
		"Artist - Title (Extended 12\" Version).mp3",
		"Artist - Title (Original 12-inch Mix).mp3",

		// === COVER / TRIBUTE CONVENTIONS ===
		"Artist - Title (originally by OtherArtist).mp3",
		"Artist - Title (Cover).mp3",
		"Artist covers OtherArtist - Title.mp3",
		"Artist - Title (OtherArtist Cover).mp3",
		"Artist - Title (Tribute to OtherArtist).mp3",

		// === COMPILATION SPECIFIC ===
		"VA - 2024 Hits Collection - 01 - Artist - Song.mp3",
		"V.A. - Collection - 01 - Artist - Song.mp3",
		"v.a. - collection - 01 - artist - song.mp3",
		"Various - Collection - 01 - Artist - Song.mp3",

		// === EPISODE / PODCAST CONVENTIONS ===
		"Show Name Ep. 042 - Topic - 2024-01-15.mp3",
		"Podcast Name - Episode 142 - Guest Name.mp3",
		"S01E03 - Title.mp3",
		"Joe Rogan Experience #2000 - Elon Musk.mp3",
		"H3 Podcast #500 - Guest Name.mp3",

		// === DATE-PREFIXED RECORDINGS ===
		"2024-01-15 Artist - Title.mp3",
		"2024-01-15 - Artist - Title.mp3",
		"[2024-01-15] Artist - Title.mp3",
		"20240115 - Artist - Title.mp3",

		// === NESTED PARENTHESES ===
		"Artist - Title (Song (From Movie)).mp3",
		"Artist - Title (Remix (Extended)).mp3",
		"Artist - (Featured B) Title (Remix).mp3",
		"Artist (Solo) - Title (Acoustic) (Live).mp3",

		// === TITLE WITH SUBTITLE COLON ===
		"Artist - Title: A Subtitle.mp3",
		"Artist - Title - The Subtitle.mp3",
		"Artist - Main Title (Subtitle).mp3",

		// === REVERSED ORDER (some users put title first) ===
		"Title - Artist.mp3",                    // ambiguous!
		"Song Name - by Artist Name.mp3",
		"Song Name by Artist.mp3",

		// === NO-SPACE-AFTER-DASH TYPOS ===
		"Artist-Title.mp3",                       // bare dash (no space)
		"Artist -Title.mp3",                      // space only before
		"Artist- Title.mp3",                      // space only after
		"The Beatles- Hey Jude.mp3",
		"Eminem-Without Me.mp3",

		// === MULTIPLE SPACES, TABS ===
		"Artist  -  Title.mp3",                   // double spaces around separator
		"Artist\t-\tTitle.mp3",                   // tabs!
		"Artist - Title.mp3",                     // single tab (rare)
		"Artist   ---   Title.mp3",               // ridiculous spacing

		// === REAL TRACK CONVENTIONS WITH LEADING ZERO ===
		"02 - Artist - Title.mp3",
		"02.Artist-Title.mp3",
		"02)Artist - Title.mp3",
		"(02) Artist - Title.mp3",
		"02 ) Artist - Title.mp3",
		"02. - Artist - Title.mp3",

		// === SINGLE-NAME ARTISTS ===
		"Cher - Believe.mp3",
		"Madonna - Vogue.mp3",
		"Drake - Hotline Bling.mp3",
		"Pink - So What.mp3",
		"Bono - Title.mp3",

		// === FILENAMES WITH MULTIPLE PERIODS ===
		"Mr. Mister - Kyrie.mp3",
		"Dr. John - Right Place Wrong Time.mp3",
		"Mr. Big - To Be With You.mp3",
		"St. Vincent - Digital Witness.mp3",

		// === EMPHASIS WITH UNDERSCORES (very old conventions) ===
		"__Artist__ - __Title__.mp3",
		"--Artist-- - --Title--.mp3",
		"==Artist== - ==Title==.mp3",
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
		// Only flag truly unrealistic chars. #, @, |, $, ! are all
		// legitimate in real artist/title names (Podcast #500, Artist@Venue,
		// Ke$ha, SAD!, "Artist | Title" separator).
		const reallyGarbage = "~=%^*`\\;"
		if strings.ContainsAny(r.Artist, reallyGarbage) {
			sb.WriteString("     ⚠ WARNING: Artist contains garbage chars!\n")
			hasIssue = true
		}
		if strings.ContainsAny(r.Title, reallyGarbage) {
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
