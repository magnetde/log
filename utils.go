package log

import "strings"

func divCeil(x int, y int) int {
	return (x + y - 1) / y
}

// PadStart konkateniert den aktuellen String mit einem weiteren String auf (bei Bedarf mehrfach),
// bis der resultierende String die angegebene Länge erreicht.
// Das Auffüllen wird ab dem Anfang der aktuellen Zeichenkette angewendet.
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
