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
	" -",
	"- ",
}

var trackPrefixRe = regexp.MustCompile(`^(\d{1,3})\s*[.\-_)\s]\s*`)

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

// Trailing ID/garbage numbers: -21498, _83621 (separator + 4+ digits at end)
var trailingIdRe = regexp.MustCompile(`[-_]\d{4,}$`)

func Parse(path string) ParseResult {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	result := ParseResult{
		FilePath: path,
		Filename: filename,
	}

	name = strings.TrimSpace(name)

	// Step 1: Strip copy suffixes (— копия, - Copy, (2), etc.)
	name = stripCopySuffix(name)

	// Step 1.5: Strip trailing garbage IDs (-21498) and track suffixes (_01)
	name = stripTrailingId(name)
	var trackFromSuffix int
	name, trackFromSuffix = extractTrackSuffix(name)

	// Step 2: Extract track number from beginning
	name, result.Track = extractTrack(name)
	if result.Track == 0 {
		result.Track = trackFromSuffix
	}

	// Step 3: Extract featured artists from parentheses (feat. X)
	name, result.FeaturedArtists = extractFeatInParens(name)

	// Step 4: Extract extras from end (Remix), [Explicit], etc.
	name, result.Extras = extractExtras(name)

	// Step 5: Split by separator into artist / title
	artist, title, found := splitBySeparator(name)

	if found {
		// Step 6: Extract any remaining inline featured artists
		artist, title, result.FeaturedArtists = extractFeaturedInline(artist, title, result.FeaturedArtists)

		result.Artist = strings.TrimSpace(artist)
		result.Title = strings.TrimSpace(title)
		result.Confidence = scoreConfidence(result)
	} else {
		// No separator found — clean up hyphenated names (police-siren → police siren)
		result.Title = cleanHyphenatedName(strings.TrimSpace(name))
		result.Confidence = Low
	}

	// Step 7: Apply title case to all-lowercase or ALL-CAPS names
	result.Artist = titleCase(result.Artist)
	result.Title = titleCase(result.Title)

	return result
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
		// m[2]:m[3] is the captured group (artist names)
		featStr := name[m[2]:m[3]]
		featured = append(featured, splitArtists(featStr)...)
		// Remove the whole (feat. ...) from the name
		name = strings.TrimSpace(name[:m[0]] + name[m[1]:])
	}
	return name, featured
}

func extractExtras(name string) (string, string) {
	var extras []string

	for {
		if m := extrasParenRe.FindStringSubmatch(name); m != nil {
			extras = append([]string{m[1]}, extras...)
			name = strings.TrimSpace(name[:extrasParenRe.FindStringIndex(name)[0]])
			continue
		}
		if m := extrasBracketRe.FindStringSubmatch(name); m != nil {
			extras = append([]string{m[1]}, extras...)
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
	// Find dashes that look like separators (letter-dash-letter)
	for i, r := range runes {
		if r != '-' {
			continue
		}
		if i == 0 || i == len(runes)-1 {
			continue
		}
		before := runes[i-1]
		after := runes[i+1]
		// Accept if at least one side has a space (looks like separator, not hyphenated word)
		if unicode.IsSpace(before) || unicode.IsSpace(after) {
			return i
		}
		// Accept bare letter-dash-letter only if it's the sole dash
		// and both sides have enough content (not "Guns-N-Roses" pattern)
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

	// Check artist field for inline featured
	for _, re := range featPatterns {
		if loc := re.FindStringIndex(artist); loc != nil {
			featPart := strings.TrimSpace(artist[loc[1]:])
			artist = strings.TrimSpace(artist[:loc[0]])
			featured = append(featured, splitArtists(featPart)...)
			break
		}
	}

	// Check title field for inline featured
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

func splitArtists(s string) []string {
	parts := []string{s}

	delimiters := []string{" & ", ", ", " vs. ", " vs "}
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

// stripTrailingId removes garbage ID numbers from end of filename (-21498, _83621).
func stripTrailingId(name string) string {
	return strings.TrimSpace(trailingIdRe.ReplaceAllString(name, ""))
}

// cleanHyphenatedName replaces hyphens with spaces when all parts between
// hyphens are lowercase (looks like word-separating hyphens, not names like Wu-Tang).
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
		// If any part has uppercase letters, keep hyphens (could be a name)
		if p != strings.ToLower(p) {
			return name
		}
	}
	// All parts lowercase — replace hyphens with spaces
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return strings.Join(cleaned, " ")
}

// titleCase capitalizes the first letter of each word.
// Only applied when the string is ALL CAPS or all lowercase (and > 3 chars).
// Short strings (1-3 chars) like "NF" or "DJ" are left unchanged.
func titleCase(s string) string {
	runes := []rune(s)
	if len(runes) <= 3 {
		return s
	}

	lower := strings.ToLower(s)
	upper := strings.ToUpper(s)

	// Only transform if uniformly cased
	if s != lower && s != upper {
		return s
	}

	words := strings.Fields(lower)
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
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
