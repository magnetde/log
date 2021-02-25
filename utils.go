package log

import "strings"

// divCeil calculates ceil(x / y)
func divCeil(x int, y int) int {
	return (x + y - 1) / y
}

// padStart pads the current string with another string (multiple times if needed)
// until the resulting string reaches the specified length.
// Padding is applied from the beginning of the current string.
func padStart(s string, l int, fill string) string {
	stringLength := len(s)

	if l <= stringLength || fill == "" {
		return s
	}

	fillLen := l - stringLength
	stringFiller := strings.Repeat(fill, divCeil(fillLen, len(fill)))

	if len(stringFiller) > fillLen {
		stringFiller = stringFiller[0:fillLen]
	}

	return stringFiller + s
}
