package parser

import (
	"regexp"
	"strings"
)

// metaJunkRe matches metadata values that are generic/useless placeholders.
var metaJunkRe = regexp.MustCompile(`(?i)^(?:` +
	`unknown(?:\s+artist)?` +
	`|track\s*#?\s*\d*` +
	`|audio\s*track\s*\d*` +
	`|untitled` +
	`|www\..+` +
	`|https?://.+` +
	`)$`)

// isJunkMetadata returns true if the metadata value is a useless placeholder or too short.
func isJunkMetadata(s string) bool {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= 1 {
		return true
	}
	return metaJunkRe.MatchString(s)
}

// CleanMetadata applies the parser's cleaning pipeline to raw metadata tag values.
// It strips junk extras, extracts featured artists, normalizes casing, etc.
// Returns cleaned artist, title, featured artists list, and extras string.
func CleanMetadata(artist, title string) (cleanArtist, cleanTitle string, featured []string, extras string) {
	cleanArtist = strings.TrimSpace(artist)
	if cleanArtist != "" {
		// Extract featured artists from parentheses: (feat. X), (ft. X)
		cleanArtist, featured = extractFeatInParens(cleanArtist)
		// Strip junk parenthesized content from artist (shouldn't be there but happens)
		cleanArtist, _ = extractExtras(cleanArtist)
		// Extract inline featured: "Artist feat. X" → "Artist", ["X"]
		var inlineFeat []string
		cleanArtist, _, inlineFeat = extractFeaturedInline(cleanArtist, "", nil)
		featured = append(featured, inlineFeat...)
		// Normalize "+" to "&"
		cleanArtist = normalizePlusInArtist(cleanArtist)
		// No titleCase — metadata casing is intentional
		// Trim garbage
		cleanArtist = trimGarbageEdges(cleanArtist)
		cleanArtist = trimOrphanDashes(cleanArtist)
		if isJunkMetadata(cleanArtist) {
			cleanArtist = ""
			featured = nil
		}
	}

	cleanTitle = strings.TrimSpace(title)
	if cleanTitle != "" {
		// Extract featured artists from title: "Song (feat. X)" → "Song", ["X"]
		var titleFeat []string
		cleanTitle, titleFeat = extractFeatInParens(cleanTitle)
		featured = append(featured, titleFeat...)
		// Extract extras, filtering junk: (Remix) kept, (320kbps) stripped
		cleanTitle, extras = extractExtras(cleanTitle)
		// Extract inline featured from title
		var inlineFeat []string
		_, cleanTitle, inlineFeat = extractFeaturedInline("", cleanTitle, nil)
		featured = append(featured, inlineFeat...)
		// Strip unbracketed converter junk ("Song Official Video" → "Song")
		cleanTitle = stripTrailingJunkPhrases(cleanTitle)
		// No titleCase — metadata casing is intentional
		// Trim garbage
		cleanTitle = trimGarbageEdges(cleanTitle)
		cleanTitle = trimOrphanDashes(cleanTitle)
		if isJunkMetadata(cleanTitle) {
			cleanTitle = ""
			extras = ""
		}
	}

	featured = dedupeArtists(featured)
	return
}
