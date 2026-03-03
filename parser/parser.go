package parser

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
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
	" -",
	"- ",
}

var trackPrefixRe = regexp.MustCompile(`^#?(\d{1,3})\s*[.\-_)\s]\s*`)

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

	// Step 0: Strip copy suffixes (— копия, - Copy, (2), etc.)
	name = stripCopySuffix(name)

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

	// Step 6: Extract track number from beginning (#01, 01., etc.)
	name, result.Track = extractTrack(name)
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
		result.Confidence = scoreConfidence(result)
	} else {
		// No separator found — clean up hyphenated names (police-siren → police siren)
		result.Title = cleanHyphenatedName(strings.TrimSpace(name))
		result.Confidence = Low

		// Heuristic: if the filename was purely underscore-delimited (VK, SoundCloud, etc.)
		// try splitting into artist / title by word count:
		//   2 words → [1] artist + [1] title
		//   3+ words → [2] artist + [rest] title
		if wasUnderscoreName {
			words := strings.Fields(result.Title)
			if len(words) >= 2 {
				splitAt := 1
				if len(words) >= 3 {
					splitAt = 2
				}
				result.Artist = strings.Join(words[:splitAt], " ")
				result.Title = strings.Join(words[splitAt:], " ")
				result.Confidence = Medium
			}
		}
	}

	// Step 12: Apply title case to all-lowercase or ALL-CAPS names
	result.Artist = titleCase(result.Artist)
	result.Title = titleCase(result.Title)

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
	loc := trackPrefixRe.FindStringIndex(name)
	if loc == nil {
		return name, 0
	}
	match := trackPrefixRe.FindStringSubmatch(name)
	if match == nil {
		return name, 0
	}
	num := 0
	for _, c := range match[1] {
		num = num*10 + int(c-'0')
	}
	return strings.TrimSpace(name[loc[1]:]), num
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

func findBareDash(name string) int {
	runes := []rune(name)
	for i, r := range runes {
		if r != '-' {
			continue
		}
		if i == 0 || i == len(runes)-1 {
			continue
		}
		before := runes[i-1]
		after := runes[i+1]
		if unicode.IsSpace(before) || unicode.IsSpace(after) {
			return i
		}
		dashCount := 0
		for _, c := range runes {
			if c == '-' {
				dashCount++
			}
		}
		if dashCount == 1 {
			return i
		}
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

// isGarbageId returns true if the string looks like a numeric/hex ID.
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

// titleCase capitalizes the first letter of each word.
// Each word is checked independently: all-lower or ALL-CAPS words get title-cased,
// known abbreviations (DJ, NF, MC, etc.) are kept ALL-CAPS,
// mixed-case words (e.g. "McDonald") are left as-is.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r := []rune(w)
		if len(r) <= 1 {
			words[i] = strings.ToUpper(w)
			continue
		}
		upper := strings.ToUpper(w)
		if abbreviations[upper] {
			words[i] = upper
			continue
		}
		lower := strings.ToLower(w)
		if w == lower || w == upper {
			r = []rune(lower)
			r[0] = unicode.ToUpper(r[0])
			words[i] = string(r)
		}
	}
	return strings.Join(words, " ")
}

func scoreConfidence(r ParseResult) Confidence {
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
