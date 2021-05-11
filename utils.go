package log

import (
	"bytes"
	"fmt"
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

// Renames all files with the least overhead.
// See https://stackoverflow.com/questions/43775524/rename-all-contents-of-directory-with-a-minimum-of-overhead
func renameAll(D map[string]string) error {
	moved := make(map[string]bool)     // set
	filenames := make(map[string]bool) // set
	tmp := getTempfile(filepath.Join(os.TempDir(), "rename"))

	rename := func(start, dest string) error {
		moved[start] = true
		return os.Rename(start, dest)
	}

	for start := range D {
		if _, ok := moved[start]; ok {
			continue
		}

		A := []string{} // List of files to rename
		p := start

		for {
			A = append(A, p)
			dest := D[p]
			if _, ok := filenames[dest]; !ok {
				break
			}

			if dest == start {
				// Found a loop
				D[tmp] = D[start]

				err := rename(start, tmp)
				if err != nil {
					return err
				}

				A[0] = tmp
				break
			}

			p = dest
		}

		A = reverse(A)
		for _, f := range A {
			err := rename(f, D[f])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getTempfile(p string) string {
	p = filepath.Clean(p)

	max := int(1 << 20)

	for i := 0; true; i++ {
		var path string
		if i == 0 {
			path = p
		} else {
			path = fmt.Sprintf("%s-%d", p, i)
		}

		if exists, err := fileExists(path); !exists && err == nil {
			return path
		} else if err != nil {
			max >>= 1
		}
	}

	return p + "-tmp"
}

func reverse(s []string) []string {
	a := make([]string, len(s))
	copy(a, s)

	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}

	return s
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
