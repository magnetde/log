package log

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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

// countLines counts the number of lines in a file.
func countLines(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

// rawfilename returns the filename without the extension.
func rawfilename(name string) string {
	return name[:len(name)-len(filepath.Ext(name))]
}

func renameAll(renames map[string]string) error {
	return errors.New("not implemented")
}

// fileExists checks if the file exists. If an error occurs when checking, it is returned.
func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		// We dont know if the file exists or not.
		// So we report this error
		return false, err
	}
}
