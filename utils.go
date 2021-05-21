package log

import (
	"bytes"
	"errors"
	"io"
	"os"
	"regexp"
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

// countLines counts the number of lines in a file.
// To count the NewLine character the function bytes.Count() is used, because it is optimized processor specific.
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

// Renames all files with the least overhead.
func renameAll(fmap map[string]string) error {
	filenames := make(map[string]bool)
	for k := range fmap {
		filenames[k] = true
	}

	for len(filenames) > 0 {
		var file string
		for k := range fmap { // getting first element from map
			file = k
			break
		}

		err := renameRecursive(fmap, filenames, file, 0)
		if err != nil {
			return err
		}
	}

	return nil
}

// renameRecursive deletes the respective file.
// If the destination filename is already in use, it recursively renames the respective destination first.
func renameRecursive(fmap map[string]string, filenames map[string]bool, file string, depth int) error {
	if depth > 16*1024 {
		return errors.New("maximum recursion depth for rotating reached")
	}

	if _, ok := filenames[fmap[file]]; ok {
		renameRecursive(fmap, filenames, fmap[file], depth+1)
	}

	err := os.Rename(file, fmap[file])
	if err != nil {
		return err
	}

	delete(fmap, file)
	delete(filenames, file)
	return nil
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
