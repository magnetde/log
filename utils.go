package serverhook

import (
	"fmt"
	"regexp"
	"strings"
)

var colorParts = []string{
	"[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[-a-zA-Z\\d\\/#&.:=?%@~_]*)*)?\u0007)",
	"(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PR-TZcf-ntqry=><~]))",
}
var colorRegex = regexp.MustCompile(strings.Join(colorParts, "|"))

// removeColors removes ANSI-colors in a string
func removeColors(s string) string {
	if colorRegex.MatchString(s) {
		return colorRegex.ReplaceAllString(s, "")
	}

	return s
}

// quoteIfNeeded adds quotation marks to the string if needed.
func quoteIfNeeded(s string) string {
	for _, ch := range s {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return fmt.Sprintf("%q", s)
		}
	}

	return s
}
