package textutil

import "strings"

// Snippet collapses whitespace and adds an ellipsis after limit runes if truncated.
func Snippet(s string, limit int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > limit {
		return string(r[:limit]) + "…"
	}
	return s
}
