package parser

import (
	"regexp"
	"strings"
)

// punctReplacer maps full-width/CJK punctuation and exotic dash variants to
// their ASCII equivalents so the rest of the pipeline only deals with one
// form. Quote characters («», “”, 「」, 『』) are intentionally NOT mapped —
// quoted-title splitting treats them as structure.
var punctReplacer = strings.NewReplacer(
	"－", "-", // full-width hyphen-minus
	"‐", "-", // U+2010 hyphen
	"‑", "-", // U+2011 non-breaking hyphen
	"‒", "-", // U+2012 figure dash
	"–", "-", // U+2013 en dash
	"—", "-", // U+2014 em dash
	"―", "-", // U+2015 horizontal bar
	"−", "-", // U+2212 minus sign
	"（", "(",
	"）", ")",
	"【", "[",
	"】", "]",
	"［", "[",
	"］", "]",
	"｛", "{",
	"｝", "}",
	"｜", "|",
	"～", "~",
	"　", " ", // ideographic space
)

func normalizePunct(name string) string {
	return punctReplacer.Replace(name)
}

// sitePrefixRe matches download-site branding at the start of a name:
// "y2mate.com - ", "[Mp3Juices.cc] ", "www.NewAlbumReleases.net_".
// The TLD list is deliberately narrow so artist names that merely look
// like domains (will.i.am, Run-D.M.C., dwt.tv) survive.
var sitePrefixRe = regexp.MustCompile(`(?i)^[\[(]?\s*(?:www\.)?[a-z0-9][a-z0-9-]{1,30}\.(?:com|net|org|io|me|cc|co|ru|su|to|ws|info|biz|site|xyz|top|club|online|pro|fm)[\])]?\s*[-_:|]*\s*`)

func stripSitePrefix(name string) string {
	stripped := sitePrefixRe.ReplaceAllString(name, "")
	if strings.TrimSpace(stripped) == "" {
		return name // the whole name was the "prefix" — keep it for manual review
	}
	return stripped
}

// leadingBracketRe captures bracketed content at the START of a name so it
// can be junk-tested: "【MV】Artist - Title" (normalized to "[MV]..."),
// "[Official Video] Artist - Title". Non-junk content is left alone.
var leadingBracketRe = regexp.MustCompile(`^[\[(]([^\])]{1,40})[\])]\s*`)

func stripLeadingJunkBrackets(name string) string {
	for {
		m := leadingBracketRe.FindStringSubmatch(name)
		if m == nil || !isJunkExtra(m[1]) {
			return name
		}
		name = strings.TrimSpace(name[len(m[0]):])
	}
}

// Junk extras: quality tags, format tags, URLs, video-platform noise.
// Anchored ^...$ — partial matches keep the extra ("8D Audio", "Sub Focus Remix").
var junkExtraRe = regexp.MustCompile(`(?i)^(?:` +
	`\d{2,4}\s*k(?:bps|b/s)?` + // 320kbps, 128kb/s
	`|\d{3}` + // bare bitrate: 128, 192, 256, 320
	`|HQ|HD|LQ|CDQ|SQ|4K|2160p|1440p|1080p|720p|480p|360p|60\s*fps` + // quality markers
	`|HQ\s+audio|high\s+quality(?:\s+audio)?|best\s+quality` +
	`|FLAC|MP3|WAV|OGG|AAC|WMA|OPUS|M4A|ALAC|AIFF` + // format tags
	`|www\..+` + // URLs
	`|https?://.+` +
	`|official(?:\s+(?:video|audio|music\s+video|lyric(?:s)?\s+video|visuali[sz]er|mv|m/v|version))?(?:\s+(?:hd|4k))?` + // official-anything
	`|music\s+video|audio(?:\s+only)?|video` +
	`|lyric(?:s)?(?:\s+video)?` +
	`|visuali[sz]er` +
	`|mv|m/v|vevo` +
	`|videoclip(?:\s+oficial)?|video\s+oficial|clip\s+officiel` +
	`|color\s+coded(?:\s+lyric(?:s)?)?(?:\s+.*)?` + // (Color Coded Lyrics Han/Rom/Eng)
	`|sub\s+(?:español|espanol|esp|eng(?:lish)?|ita(?:liano)?|indo(?:nesia)?|thai|fr(?:ench)?|rus(?:sian)?)` +
	`|subtitulado|legendado` +
	`|ncs(?:\s+release)?|no\s+copyright(?:\s+(?:sounds?|music))?|copyright\s+free|royalty\s+free` +
	`|full\s+(?:version|album)` +
	`|free\s+download` +
	`|bonus\s+track` +
	`|out\s+now|premiere|new\s+(?:song|track|video|music)(?:\s+\d{4})?` +
	`)$`)

// isJunkExtra returns true if the extra is a quality tag, format tag, URL,
// platform noise, or a yt-dlp video-id suffix.
func isJunkExtra(s string) bool {
	s = strings.TrimSpace(s)
	return junkExtraRe.MatchString(s) || looksLikeYouTubeID(s)
}

// looksLikeYouTubeID reports whether s looks like an 11-char YouTube video id
// ("dQw4w9WgXcQ") — yt-dlp's default output template appends " [id]" to the
// title. Requires a digit plus either -/_ or inner mixed case, so real extras
// of similar shape ("Instrumental" is 12 chars, "Slowed-Down" has no digit)
// survive.
func looksLikeYouTubeID(s string) bool {
	if len(s) != 11 {
		return false
	}
	hasDigit, hasLower, hasInnerUpper, hasDashUnderscore := false, false, false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			hasDigit = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			if i > 0 {
				hasInnerUpper = true
			}
		case c == '-' || c == '_':
			hasDashUnderscore = true
		default:
			return false
		}
	}
	return hasDigit && (hasDashUnderscore || (hasLower && hasInnerUpper))
}

// ytIDSuffixRe captures a bracketed 11-char candidate at the end of a name.
// Must run BEFORE underscore replacement — IDs often contain underscores.
var ytIDSuffixRe = regexp.MustCompile(`\s*\[([A-Za-z0-9_-]{11})\]$`)

// stripYouTubeIDSuffix removes yt-dlp's " [videoid]" suffix
// ("Title [dQw4w9WgXcQ]" → "Title").
func stripYouTubeIDSuffix(name string) string {
	if m := ytIDSuffixRe.FindStringSubmatch(name); m != nil && looksLikeYouTubeID(m[1]) {
		return strings.TrimSpace(ytIDSuffixRe.ReplaceAllString(name, ""))
	}
	return name
}

// trailingJunkPhrases are junk suffixes YouTube converters leave OUTSIDE any
// brackets ("Artist - Title Official Video.mp3"). Multi-word phrases first;
// bare generic words ("video", "audio") are deliberately absent so real
// titles like "Video Games" survive.
var trailingJunkPhrases = []string{
	"official music video",
	"official lyric video",
	"official lyrics video",
	"official video hd",
	"official video",
	"official audio",
	"official visualizer",
	"official mv",
	"music video",
	"lyric video",
	"lyrics video",
	"lyrics",
	"official",
	"hd", "hq", "4k", "mv", "m/v",
}

// stripTrailingJunkPhrases removes unbracketed junk phrases from the end of a
// title, always leaving at least one word behind.
func stripTrailingJunkPhrases(s string) string {
	words := strings.Fields(s)
	changed := true
	for changed {
		changed = false
		for _, p := range trailingJunkPhrases {
			pw := strings.Fields(p)
			if len(words) <= len(pw) {
				continue
			}
			match := true
			for i := range pw {
				if !strings.EqualFold(words[len(words)-len(pw)+i], pw[i]) {
					match = false
					break
				}
			}
			if match {
				words = words[:len(words)-len(pw)]
				changed = true
				break
			}
		}
	}
	out := strings.Join(words, " ")
	return trimOrphanDashes(out)
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

// Multiple dashes/equals → single separator
var multiDashRe = regexp.MustCompile(`\s*[-=]{2,}\s*`)

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

// Windows/macOS copy suffixes to strip
var copySuffixRe = regexp.MustCompile(`(?i)\s*[-—–]\s*(?:копия|copy|copia|copie|kopie)\s*(?:\(\d+\))?\s*$`)
var copyNumberRe = regexp.MustCompile(`\s*\(\d+\)\s*$`)

func stripCopySuffix(name string) string {
	name = copySuffixRe.ReplaceAllString(name, "")
	name = copyNumberRe.ReplaceAllString(name, "")
	return strings.TrimSpace(name)
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
