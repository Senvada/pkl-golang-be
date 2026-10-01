package utils

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	nonAlphaNumericRegex = regexp.MustCompile(`[^a-z0-9]+`)
	multipleDashRegex    = regexp.MustCompile(`-+`)
)

func Slugify(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, _ := transform.String(t, s)

	lower := strings.ToLower(normalized)
	clean := nonAlphaNumericRegex.ReplaceAllString(lower, "-")
	clean = multipleDashRegex.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-")

	return clean
}
