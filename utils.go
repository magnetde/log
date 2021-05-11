package log

import (
	"bytes"
	"errors"
	"io"
	"os"
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

// Renames all files with the least overhead.
// See https://stackoverflow.com/questions/43775524/rename-all-contents-of-directory-with-a-minimum-of-overhead
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

func renameRecursive(fmap map[string]string, filenames map[string]bool, file string, depth int) error {
	if depth > 8192 {
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
