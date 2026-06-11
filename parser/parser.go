package parser

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Confidence int

const (
	Low Confidence = iota
	Medium
	High
)

func (c Confidence) String() string {
	switch c {
	case High:
		return "high"
	case Medium:
		return "medium"
	default:
		return "low"
	}
}

type ParseResult struct {
	FilePath        string     `json:"filePath"`
	Filename        string     `json:"filename"`
	Artist          string     `json:"artist"`
	Title           string     `json:"title"`
	Track           int        `json:"track"`
	FeaturedArtists []string   `json:"featuredArtists,omitempty"`
	Extras          string     `json:"extras,omitempty"`
	Confidence      Confidence `json:"confidence"`
}

// Separators tried in priority order for splitting artist / title.
// Em/en dashes are normalized to "-" before splitting, so only ASCII
// dash forms appear here.
var separators = []string{
	" - ",
	"_-_",
	"+-+",
	" | ", // pipe-with-spaces (rare but real, e.g. "Artist | Title.mp3")
	" • ", // bullet
	" → ", // arrow
	" ► ", // pointer
	" -",
	"- ",
}

// Track prefixes accepted:
//   "#NN " (explicit # marker, then digits, then space)
//   "NN.", "NN-", "NN_", "NN)" (digits + punctuation, space optional)
// Bare "NN " (digits + space alone) is NOT a track prefix — too ambiguous
// with band names like "50 Cent", "3 Doors Down", "4 Non Blondes".
var trackPrefixHashRe = regexp.MustCompile(`^#(\d{1,3})\s+`)
var trackPrefixPunctRe = regexp.MustCompile(`^(\d{1,3})\s*[.\-_)]\s*`)

// Extras at end of string: (Remix), [Explicit], etc.
var extrasParenRe = regexp.MustCompile(`\s*\(([^)]+)\)\s*$`)
var extrasBracketRe = regexp.MustCompile(`\s*\[([^\]]+)\]\s*$`)

// Featured artist patterns inside parentheses: (feat. X), (ft. X), (with X).
// "(with X)" is the Spotify collab style — treated as featuring.
var featInParensRe = regexp.MustCompile(`(?i)\((?:feat\.?|ft\.?|featuring|with)\s+([^)]+)\)`)

// Featured artist patterns inline
var featPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\s+feat\.\s+`),
	regexp.MustCompile(`(?i)\s+featuring\s+`),
	regexp.MustCompile(`(?i)\s+feat\s+`),
	regexp.MustCompile(`(?i)\s+ft\.\s+`),
	regexp.MustCompile(`(?i)\s+ft\s+`),
}

// Trailing track suffix: _01, _02, _1 (underscore + 1-3 digits at end)
var trackSuffixRe = regexp.MustCompile(`_(\d{1,3})$`)

// Disc prefix at start: "Disc 1 - ", "CD2 - ", "Disc 1 Track 03 - ".
// Captures the inner track number (if present) so it isn't lost.
var discPrefixRe = regexp.MustCompile(`(?i)^(?:disc|cd)\s*\d+\s*(?:track\s*(\d+))?\s*[-_.]\s*`)

// Date prefix at start: "2024-01-15 - ", "[2024-01-15] Artist", "20240115 ".
// Requires a real-looking date: 19xx/20xx year + month + day, either
// hyphen/underscore-separated OR packed 8-digit form. Trailing match
// accepts whitespace alone OR explicit punctuation — but the date itself
// must look like a date (year in 1900-2099 range, separators), so we
// don't eat random 8-digit garbage like "12345678".
var datePrefixRe = regexp.MustCompile(`^\[?(?:(?:19|20)\d{2}[-_]\d{2}[-_]\d{2}|(?:19|20)\d{6})\]?(?:\s+|\s*[-_.]\s*)`)

func Parse(path string) ParseResult {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	result := ParseResult{
		FilePath: path,
		Filename: filename,
	}

	name = strings.TrimSpace(name)

	// Step 0a: Normalize full-width/CJK punctuation (【】→[], （）→()) and
	// exotic dash variants (em/en dash, U+2212 minus, full-width hyphen) → "-".
	name = normalizePunct(name)

	// Step 0b: Unwrap whole-name parens/brackets like "(Artist - Title).mp3"
	// or "[Artist - Title].mp3". Without this, extractExtras would eat the
	// whole content as a single "extra" and leave the name empty.
	name = unwrapWholeNameBrackets(name)

	// Step 0c: Strip download-site branding ("y2mate.com - ", "[Mp3Juices.cc] ").
	name = stripSitePrefix(name)

	// Step 0d: Strip leading bracketed junk ("[MV]", "[Official Video]").
	name = stripLeadingJunkBrackets(name)

	// Step 0e: Strip copy suffixes (— копия, - Copy, (2), etc.)
	name = stripCopySuffix(name)

	// Step 0f: Strip yt-dlp "[videoid]" suffix while underscores are intact.
	name = stripYouTubeIDSuffix(name)

	// Strip disc prefix (Disc 1 - , CD2 - , Disc 1 Track 03 - ).
	// If the prefix contains an embedded track number ("Disc 1 Track 03"),
	// capture it so we don't lose the track count.
	if m := discPrefixRe.FindStringSubmatch(name); m != nil {
		if m[1] != "" {
			n := 0
			for _, c := range m[1] {
				n = n*10 + int(c-'0')
			}
			result.Track = n
		}
		name = name[len(m[0]):]
	}

	// Strip leading date prefix ("2024-01-15 - ", "20240115 ", "[2024-01-15] ")
	// commonly seen on voice memos and podcast filenames.
	name = datePrefixRe.ReplaceAllString(name, "")

	// Extract track prefix EARLY, while # and _ separators are still intact.
	// (Later steps strip # as a garbage char and replace _ with spaces.)
	// Skip if a disc prefix already supplied the track number.
	if result.Track == 0 {
		name, result.Track = extractTrack(name)
	}

	// Step 1: Extract track suffix (_01) before underscore replacement
	var trackFromSuffix int
	name, trackFromSuffix = extractTrackSuffix(name)

	// Step 2: Strip trailing garbage IDs (-21498, _713893675, _9dcc104f)
	name = stripTrailingIds(name)

	// Step 3: Replace underscores with spaces (VK, SoundCloud, etc.)
	nameBeforeUnderscore := name
	name = replaceUnderscores(name)
	wasUnderscoreName := name != nameBeforeUnderscore && !strings.Contains(nameBeforeUnderscore, " ")

	// Step 4: Normalize garbage characters, compound separators, curly braces
	name = cleanupName(name)

	// Step 5: Strip any remaining space-separated trailing garbage
	name = stripTrailingGarbageWords(name)

	// Step 6: If track wasn't found at the early extract pass, try again now
	// (handles cases where cleanup exposed a track marker).
	if result.Track == 0 {
		name, result.Track = extractTrack(name)
	}
	if result.Track == 0 {
		result.Track = trackFromSuffix
	}

	// Step 7: Extract featured artists from parentheses (feat. X)
	name, result.FeaturedArtists = extractFeatInParens(name)

	// Step 8: Extract extras from end (Remix), [Explicit], etc. — filters junk
	name, result.Extras = extractExtras(name)

	// Step 9: Split by separator into artist / title
	artist, title, found := splitBySeparator(name)

	if found {
		// Step 10: Extract any remaining inline featured artists
		artist, title, result.FeaturedArtists = extractFeaturedInline(artist, title, result.FeaturedArtists)

		// Step 11: Normalize "+" to "&" in artist (Artist1 + Artist2 → Artist1 & Artist2)
		artist = normalizePlusInArtist(artist)

		// Step 12: Unwrap quoted segments («Title» → Title) and strip
		// unbracketed trailing junk ("Title Official Video" → "Title").
		artist = unwrapQuotes(strings.TrimSpace(artist))
		title = unwrapQuotes(strings.TrimSpace(title))
		title = stripTrailingJunkPhrases(title)

		result.Artist = strings.TrimSpace(artist)
		result.Title = strings.TrimSpace(title)
		result.Confidence = ScoreConfidence(result)
	} else {
		// No separator found — check if it's a platform-style hyphenated name
		wasHyphenatedName := isAllLowerHyphenated(strings.TrimSpace(name))

		// Clean up hyphenated names (police-siren → police siren)
		result.Title = cleanHyphenatedName(strings.TrimSpace(name))
		result.Confidence = Low

		// Heuristic: if the filename was purely underscore-delimited (VK, SoundCloud, etc.)
		// or all-lowercase-hyphenated (Bandcamp, SoundCloud, etc.),
		// split first segment(s) as artist, rest as title.
		if wasUnderscoreName || wasHyphenatedName {
			words := strings.Fields(result.Title)
			if len(words) >= 2 {
				splitAt := 1
				if len(words) >= 3 && wasUnderscoreName {
					splitAt = 2
				}
				result.Artist = strings.Join(words[:splitAt], " ")
				result.Title = strings.Join(words[splitAt:], " ")
			}
		}

		result.Title = stripTrailingJunkPhrases(result.Title)
	}

	// Step 13: Apply title case to all-lowercase or ALL-CAPS names
	result.Artist = TitleCase(result.Artist)
	result.Title = TitleCase(result.Title)

	// Final cleanup: trim any stray garbage from edges + orphan dashes
	result.Artist = trimGarbageEdges(result.Artist)
	result.Title = trimGarbageEdges(result.Title)
	result.Artist = trimOrphanDashes(result.Artist)
	result.Title = trimOrphanDashes(result.Title)

	// Featured artists collected from several places (parens in artist,
	// parens in title, inline feat) can repeat — keep first occurrence.
	result.FeaturedArtists = dedupeArtists(result.FeaturedArtists)

	return result
}

func extractTrack(name string) (string, int) {
	for _, re := range []*regexp.Regexp{trackPrefixHashRe, trackPrefixPunctRe} {
		loc := re.FindStringIndex(name)
		if loc == nil {
			continue
		}
		match := re.FindStringSubmatch(name)
		if match == nil {
			continue
		}
		num := 0
		for _, c := range match[1] {
			num = num*10 + int(c-'0')
		}
		return strings.TrimSpace(name[loc[1]:]), num
	}
	return name, 0
}

func extractFeatInParens(name string) (string, []string) {
	var featured []string
	for {
		m := featInParensRe.FindStringSubmatchIndex(name)
		if m == nil {
			break
		}
		featStr := name[m[2]:m[3]]
		featured = append(featured, splitArtists(featStr)...)
		name = strings.TrimSpace(name[:m[0]] + name[m[1]:])
	}
	return name, featured
}

func extractExtras(name string) (string, string) {
	var extras []string

	for {
		if m := extrasParenRe.FindStringSubmatch(name); m != nil {
			extra := strings.TrimSpace(m[1])
			if !isJunkExtra(extra) {
				extras = append([]string{extra}, extras...)
			}
			name = strings.TrimSpace(name[:extrasParenRe.FindStringIndex(name)[0]])
			continue
		}
		if m := extrasBracketRe.FindStringSubmatch(name); m != nil {
			extra := strings.TrimSpace(m[1])
			if !isJunkExtra(extra) {
				extras = append([]string{extra}, extras...)
			}
			name = strings.TrimSpace(name[:extrasBracketRe.FindStringIndex(name)[0]])
			continue
		}
		break
	}

	return name, strings.Join(extras, ", ")
}

func splitBySeparator(name string) (artist, title string, found bool) {
	for _, sep := range separators {
		idx := strings.Index(name, sep)
		if idx > 0 {
			left := strings.TrimSpace(name[:idx])
			right := strings.TrimSpace(name[idx+len(sep):])
			if left != "" && right != "" {
				return left, right, true
			}
		}
	}

	// Quoted title: "Artist «Title»" — explicit structure, beats bare-dash
	// guessing (e.g. "AC-DC «Back In Black»" must not split on the hyphen).
	if a, t, ok := splitByQuotedTitle(name); ok {
		return a, t, true
	}

	// Last resort: bare dash — use first one that has a letter/digit on each side
	if idx := findBareDash(name); idx > 0 {
		left := strings.TrimSpace(name[:idx])
		right := strings.TrimSpace(name[idx+1:])
		if left != "" && right != "" {
			return left, right, true
		}
	}

	return "", name, false
}

// findBareDash returns the BYTE index of the first dash usable as a
// separator. Callers slice by byte index, so the index must NOT be a
// rune index — that mangles UTF-8 inside multi-byte strings like
// "Артист-Название" (which would split mid-rune).
func findBareDash(name string) int {
	runes := []rune(name)
	dashCount := 0
	for _, c := range runes {
		if c == '-' {
			dashCount++
		}
	}

	byteIdx := 0
	for i, r := range runes {
		if r == '-' {
			if i > 0 && i < len(runes)-1 {
				before := runes[i-1]
				after := runes[i+1]
				if unicode.IsSpace(before) || unicode.IsSpace(after) || dashCount == 1 {
					return byteIdx
				}
			}
		}
		byteIdx += utf8.RuneLen(r)
	}
	return -1
}

// quotePairs are opening/closing quote characters that mark the title in
// "Artist «Title»"-style filenames: Russian guillemets, curly double quotes,
// German low-high quotes, CJK corner brackets, straight quotes.
var quotePairs = [][2]rune{
	{'«', '»'},
	{'“', '”'},
	{'„', '“'},
	{'「', '」'},
	{'『', '』'},
	{'"', '"'},
}

// splitByQuotedTitle splits "Artist «Title»" → ("Artist", "Title").
// The quoted segment must close the name and the artist part must be non-empty.
func splitByQuotedTitle(name string) (artist, title string, ok bool) {
	runes := []rune(name)
	if len(runes) < 3 {
		return "", "", false
	}
	last := runes[len(runes)-1]
	for _, p := range quotePairs {
		if last != p[1] {
			continue
		}
		openIdx := -1
		for i, r := range runes[:len(runes)-1] {
			if r == p[0] {
				openIdx = i
				break
			}
		}
		if openIdx <= 0 {
			continue
		}
		a := strings.TrimSpace(string(runes[:openIdx]))
		t := strings.TrimSpace(string(runes[openIdx+1 : len(runes)-1]))
		if a != "" && t != "" {
			return a, t, true
		}
	}
	return "", "", false
}

// unwrapQuotes strips surrounding quote pairs: "«Title»" → "Title".
func unwrapQuotes(s string) string {
	for {
		runes := []rune(s)
		if len(runes) < 2 {
			return s
		}
		matched := false
		for _, p := range quotePairs {
			if runes[0] == p[0] && runes[len(runes)-1] == p[1] {
				s = strings.TrimSpace(string(runes[1 : len(runes)-1]))
				matched = true
				break
			}
		}
		if !matched {
			return s
		}
	}
}

func extractFeaturedInline(artist, title string, existing []string) (string, string, []string) {
	featured := append([]string{}, existing...)

	for _, re := range featPatterns {
		if loc := re.FindStringIndex(artist); loc != nil {
			featPart := strings.TrimSpace(artist[loc[1]:])
			artist = strings.TrimSpace(artist[:loc[0]])
			featured = append(featured, splitArtists(featPart)...)
			break
		}
	}

	for _, re := range featPatterns {
		if loc := re.FindStringIndex(title); loc != nil {
			featPart := strings.TrimSpace(title[loc[1]:])
			title = strings.TrimSpace(title[:loc[0]])
			featured = append(featured, splitArtists(featPart)...)
			break
		}
	}

	return artist, title, featured
}

// normalizePlusInArtist converts " + " to " & " in artist names.
func normalizePlusInArtist(artist string) string {
	return strings.ReplaceAll(artist, " + ", " & ")
}

func splitArtists(s string) []string {
	parts := []string{s}

	delimiters := []string{" & ", " + ", ", ", " vs. ", " vs "}
	for _, d := range delimiters {
		var next []string
		for _, p := range parts {
			split := strings.Split(p, d)
			next = append(next, split...)
		}
		parts = next
	}

	// Handle " x " carefully: only if both sides start with uppercase
	var final []string
	xRe := regexp.MustCompile(`\s+x\s+`)
	for _, p := range parts {
		if loc := xRe.FindStringIndex(p); loc != nil {
			left := strings.TrimSpace(p[:loc[0]])
			right := strings.TrimSpace(p[loc[1]:])
			if len(left) > 0 && len(right) > 0 && unicode.IsUpper([]rune(left)[0]) && unicode.IsUpper([]rune(right)[0]) {
				final = append(final, left, right)
				continue
			}
		}
		final = append(final, p)
	}

	var result []string
	for _, p := range final {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// dedupeArtists removes case-insensitive duplicates, keeping first occurrence.
func dedupeArtists(artists []string) []string {
	if len(artists) < 2 {
		return artists
	}
	seen := make(map[string]bool, len(artists))
	var out []string
	for _, a := range artists {
		key := strings.ToLower(strings.TrimSpace(a))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, a)
	}
	return out
}

// unwrapWholeNameBrackets removes surrounding parens/brackets when the
// entire content is wrapped in them. Handles "(Artist - Title)" → "Artist - Title"
// and "[Artist - Title]" → "Artist - Title". Nested wraps are unwrapped too.
func unwrapWholeNameBrackets(name string) string {
	for {
		name = strings.TrimSpace(name)
		if len(name) < 2 {
			return name
		}
		first := name[0]
		last := name[len(name)-1]
		if (first == '(' && last == ')') || (first == '[' && last == ']') {
			inner := name[1 : len(name)-1]
			// Sanity: only unwrap if inner doesn't have its own unbalanced opener.
			if !strings.ContainsAny(inner[:1], "([") || balanced(inner) {
				name = inner
				continue
			}
		}
		return name
	}
}

func balanced(s string) bool {
	parens, brackets := 0, 0
	for _, r := range s {
		switch r {
		case '(':
			parens++
		case ')':
			parens--
			if parens < 0 {
				return false
			}
		case '[':
			brackets++
		case ']':
			brackets--
			if brackets < 0 {
				return false
			}
		}
	}
	return parens == 0 && brackets == 0
}

// extractTrackSuffix strips trailing _01, _02 etc. and returns the track number.
func extractTrackSuffix(name string) (string, int) {
	m := trackSuffixRe.FindStringSubmatch(name)
	if m == nil {
		return name, 0
	}
	num := 0
	for _, c := range m[1] {
		num = num*10 + int(c-'0')
	}
	return strings.TrimSpace(trackSuffixRe.ReplaceAllString(name, "")), num
}

// isAllLowerHyphenated returns true if the name is all-lowercase with hyphens
// and at least 2 segments (typical platform-generated filename: artist-song-title).
func isAllLowerHyphenated(name string) bool {
	if !strings.Contains(name, "-") {
		return false
	}
	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p != strings.ToLower(p) {
			return false
		}
	}
	return true
}

// cleanHyphenatedName replaces hyphens with spaces when all parts are lowercase.
func cleanHyphenatedName(name string) string {
	if !strings.Contains(name, "-") {
		return name
	}
	parts := strings.Split(name, "-")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p != strings.ToLower(p) {
			return name
		}
	}
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return strings.Join(cleaned, " ")
}

func ScoreConfidence(r ParseResult) Confidence {
	if r.Artist == "" || r.Title == "" {
		return Low
	}

	artist := r.Artist
	title := r.Title

	if len([]rune(artist)) <= 1 || len([]rune(title)) <= 1 {
		return Low
	}

	// An artist with no letters at all (digits, punctuation) is almost
	// certainly a mis-split ("2024 - Title") — flag for review.
	if !hasLetter(artist) {
		return Low
	}

	if len(artist) > 3 && (artist == strings.ToUpper(artist) || artist == strings.ToLower(artist)) {
		return Medium
	}
	if len(title) > 3 && (title == strings.ToUpper(title) || title == strings.ToLower(title)) {
		return Medium
	}

	return High
}
