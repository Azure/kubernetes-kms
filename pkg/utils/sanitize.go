package utils

import "strings"

// SanitizeString returns a string that does not have white spaces and double quotes.
func SanitizeString(s string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "\""))
}
