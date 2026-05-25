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

// Separators tried in priority order for splitting artist / title
var separators = []string{
	" - ",
	" — ",
	" – ",
	"_-_",
	"+-+",
	" | ",  // pipe-with-spaces (rare but real, e.g. "Artist | Title.mp3")
	" • ",  // bullet
	" → ",  // arrow
	" ► ",  // pointer
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

// Featured artist patterns inside parentheses: (feat. X), (ft. X)
var featInParensRe = regexp.MustCompile(`(?i)\((?:feat\.?|ft\.?|featuring)\s+([^)]+)\)`)

// Featured artist patterns inline
var featPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\s+feat\.\s+`),
	regexp.MustCompile(`(?i)\s+featuring\s+`),
	regexp.MustCompile(`(?i)\s+feat\s+`),
	regexp.MustCompile(`(?i)\s+ft\.\s+`),
	regexp.MustCompile(`(?i)\s+ft\s+`),
}

// Windows/macOS copy suffixes to strip
var copySuffixRe = regexp.MustCompile(`(?i)\s*[-—–]\s*(?:копия|copy|copia|copie|kopie)\s*(?:\(\d+\))?\s*$`)
var copyNumberRe = regexp.MustCompile(`\s*\(\d+\)\s*$`)

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

// Multiple dashes/equals → single separator
var multiDashRe = regexp.MustCompile(`\s*[-=]{2,}\s*`)

// Junk extras: quality tags, format tags, URLs, video tags
var junkExtraRe = regexp.MustCompile(`(?i)^(?:` +
	`\d{2,4}\s*k(?:bps|b/s)?` + // 320kbps, 128kb/s
	`|\d{3}` + // bare bitrate: 128, 192, 256, 320
	`|HQ|HD|LQ|CDQ|SQ` + // quality markers
	`|FLAC|MP3|WAV|OGG|AAC|WMA|OPUS|M4A|ALAC|AIFF` + // format tags
	`|www\..+` + // URLs
	`|https?://.+` +
	`|official\s+(?:video|audio|music\s+video)` + // video tags
	`|music\s+video|lyric(?:s)?\s+video|audio\s+only` +
	`|full\s+(?:version|album)` +
	`|free\s+download` +
	`|bonus\s+track` +
	`)$`)

func Parse(path string) ParseResult {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	result := ParseResult{
		FilePath: path,
		Filename: filename,
	}

	name = strings.TrimSpace(name)

	// Step 0: Unwrap whole-name parens/brackets like "(Artist - Title).mp3"
	// or "[Artist - Title].mp3". Without this, extractExtras would eat the
	// whole content as a single "extra" and leave the name empty.
	name = unwrapWholeNameBrackets(name)

	// Step 0: Strip copy suffixes (— копия, - Copy, (2), etc.)
	name = stripCopySuffix(name)

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
	}

	// Step 12: Apply title case to all-lowercase or ALL-CAPS names
	result.Artist = TitleCase(result.Artist)
	result.Title = TitleCase(result.Title)

	// Final cleanup: trim any stray garbage from edges + orphan dashes
	result.Artist = trimGarbageEdges(result.Artist)
	result.Title = trimGarbageEdges(result.Title)
	result.Artist = trimOrphanDashes(result.Artist)
	result.Title = trimOrphanDashes(result.Title)

	return result
}

// cleanupName normalizes compound separators, curly braces, and strips garbage characters.
func cleanupName(name string) string {
	// Curly braces → parentheses (for extras extraction)
	name = strings.ReplaceAll(name, "{", "(")
	name = strings.ReplaceAll(name, "}", ")")

	// Multiple dashes or equals → single separator
	name = multiDashRe.ReplaceAllString(name, " - ")

	// Strip leading/trailing garbage characters
	name = trimGarbageEdges(name)

	// Collapse multiple spaces
	name = collapseSpaces(name)

	return strings.TrimSpace(name)
}

// trimGarbageEdges strips leading/trailing non-meaningful characters.
func trimGarbageEdges(name string) string {
	runes := []rune(name)

	// Trim from start
	start := 0
	for start < len(runes) && isGarbageChar(runes[start]) {
		start++
	}

	// Trim from end
	end := len(runes)
	for end > start && isGarbageChar(runes[end-1]) {
		end--
	}

	return strings.TrimSpace(string(runes[start:end]))
}

func isGarbageChar(r rune) bool {
	switch r {
	case '~', '#', '@', '=', '%', '^', '*', '`', '|', '\\', ';':
		// Note: '!' and '$' are NOT garbage — used in real names (SAD!, Ke$ha, A$AP Rocky)
		return true
	}
	return false
}

// trimOrphanDashes removes leading/trailing dashes that aren't part of hyphenated names.
// "- Artist" → "Artist", "Title -" → "Title", "Jay-Z" → "Jay-Z" (untouched)
func trimOrphanDashes(s string) string {
	s = strings.TrimSpace(s)
	// Strip leading dashes (possibly followed by space)
	for len(s) > 0 && s[0] == '-' {
		s = strings.TrimSpace(s[1:])
	}
	// Strip trailing dashes — but NOT if attached to a word (like "Jay-Z")
	for len(s) > 0 && s[len(s)-1] == '-' {
		if len(s) >= 2 && s[len(s)-2] != ' ' {
			break // dash is attached to a word, keep it
		}
		s = strings.TrimSpace(s[:len(s)-1])
	}
	return s
}

func collapseSpaces(name string) string {
	for strings.Contains(name, "  ") {
		name = strings.ReplaceAll(name, "  ", " ")
	}
	return name
}

func stripCopySuffix(name string) string {
	name = copySuffixRe.ReplaceAllString(name, "")
	name = copyNumberRe.ReplaceAllString(name, "")
	return strings.TrimSpace(name)
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

// isJunkExtra returns true if the extra is a quality tag, format tag, URL, etc.
func isJunkExtra(s string) bool {
	return junkExtraRe.MatchString(strings.TrimSpace(s))
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

// stripTrailingIds removes trailing garbage IDs from end of filename.
func stripTrailingIds(name string) string {
	for {
		idx := strings.LastIndexAny(name, "-_")
		if idx <= 0 {
			break
		}
		if name[idx-1] == ' ' {
			break
		}
		segment := strings.TrimSpace(name[idx+1:])
		if segment == "" {
			break
		}
		if isGarbageId(segment) {
			name = strings.TrimSpace(name[:idx])
		} else {
			break
		}
	}
	return name
}

// isGarbageId returns true if the string looks like a numeric/hex ID
// from a streaming platform (VK, SoundCloud, etc.).
//
// Exception: 4-digit numbers in the year range (1000-2999) are treated
// as legitimate content — many song/album titles are years
// (Prince "1999", Bowling for Soup "1985", Tchaikovsky "1812 Overture").
// Longer all-digit strings (5+) and 4-digit non-year numbers are still
// considered garbage.
func isGarbageId(s string) bool {
	if len(s) == 0 {
		return false
	}
	if len(s) >= 4 {
		allDigits := true
		for _, c := range s {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			// Spare 4-digit years from garbage classification.
			if len(s) == 4 {
				year := (int(s[0]-'0') * 1000) + (int(s[1]-'0') * 100) +
					(int(s[2]-'0') * 10) + int(s[3]-'0')
				if year >= 1000 && year <= 2999 {
					return false
				}
			}
			return true
		}
	}
	if len(s) >= 8 {
		allHex := true
		hasDigit := false
		for _, c := range s {
			if c >= '0' && c <= '9' {
				hasDigit = true
			} else if (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
				// hex letter
			} else {
				allHex = false
				break
			}
		}
		if allHex && hasDigit {
			return true
		}
	}
	return false
}

// replaceUnderscores converts underscores to spaces.
func replaceUnderscores(name string) string {
	name = strings.ReplaceAll(name, "_", " ")
	return strings.TrimSpace(collapseSpaces(name))
}

// stripTrailingGarbageWords removes space-separated trailing words that look like IDs.
func stripTrailingGarbageWords(name string) string {
	words := strings.Fields(name)
	for len(words) > 1 {
		if isGarbageId(words[len(words)-1]) {
			words = words[:len(words)-1]
		} else {
			break
		}
	}
	return strings.Join(words, " ")
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

// Known abbreviations that should stay ALL CAPS.
var abbreviations = map[string]bool{
	"DJ": true, "MC": true, "NF": true, "ID": true, "TV": true, "OK": true,
	"EP": true, "LP": true, "UK": true, "US": true, "EU": true, "AC": true,
	"DC": true, "VR": true, "AI": true, "XL": true, "BTS": true, "EDM": true,
}

// Roman numerals up to XX — common in classical track titles
// (e.g. "Symphony No. 9 - IV. Ode to Joy").
var romanNumerals = map[string]bool{
	"I": true, "II": true, "III": true, "IV": true, "V": true, "VI": true,
	"VII": true, "VIII": true, "IX": true, "X": true, "XI": true, "XII": true,
	"XIII": true, "XIV": true, "XV": true, "XVI": true, "XVII": true,
	"XVIII": true, "XIX": true, "XX": true,
}

// hasLetter reports whether the string contains at least one Unicode letter.
func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// hasNonLetter reports whether the string contains at least one
// non-letter, non-space rune (digit, dot, dollar, etc.).
func hasNonLetter(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

// titleCase capitalizes the first letter of each word.
// Preservation rules (in order):
//   - Roman numerals (IV, V, IX, ...) stay UPPER
//   - Known abbreviations (DJ, NF, BTS, ...) stay UPPER
//   - All-UPPER words containing non-letter chars (P.O.D., A$AP, 3OH!3) are preserved
//   - All-lower words containing dots (will.i.am) are preserved
//   - Mixed-case words (Daft Punk, McDonald) are left as-is
//   - Plain all-lower or all-UPPER words → Title Case
//   - Words starting with a non-letter capitalize the first LETTER (e.g. "$abc" → "$Abc")
func TitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r := []rune(w)
		if len(r) <= 1 {
			words[i] = strings.ToUpper(w)
			continue
		}
		upper := strings.ToUpper(w)
		if romanNumerals[upper] && w == upper {
			words[i] = upper
			continue
		}
		if abbreviations[upper] {
			words[i] = upper
			continue
		}
		lower := strings.ToLower(w)
		// Stylized lowercase with dots (will.i.am, dwt.tv) — preserve.
		if w == lower && hasLetter(w) && strings.Count(w, ".") >= 2 {
			continue
		}
		// All-UPPER with non-letter chars (initialisms like P.O.D., M.I.A.,
		// stylized names like A$AP, 3OH!3) — preserve.
		if w == upper && hasNonLetter(w) && hasLetter(w) {
			continue
		}
		if w == lower || w == upper {
			lowered := []rune(lower)
			// Capitalize the first LETTER, not just the first rune
			// (so "$abc" → "$Abc", "1985" → "1985" unchanged).
			for j, c := range lowered {
				if unicode.IsLetter(c) {
					lowered[j] = unicode.ToUpper(c)
					break
				}
			}
			words[i] = string(lowered)
		}
	}
	return strings.Join(words, " ")
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

	if len(artist) > 3 && (artist == strings.ToUpper(artist) || artist == strings.ToLower(artist)) {
		return Medium
	}
	if len(title) > 3 && (title == strings.ToUpper(title) || title == strings.ToLower(title)) {
		return Medium
	}

	return High
}
