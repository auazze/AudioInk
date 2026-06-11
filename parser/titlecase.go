package parser

import (
	"strings"
	"unicode"
)

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

// TitleCase capitalizes the first letter of each word.
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
