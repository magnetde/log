package log

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

var logRegex = regexp.MustCompile(" ?[(.+)] ([(.+)])? .+")

func TestConsole(t *testing.T) {
	var buf bytes.Buffer

	Init(&ConsoleTransporter{
		Colors: false,
		Output: &buf,
	})

	Trace("test trace")
	Debug("test debug")
	Info("test info")
	Warn("test warn")
	Error("test error")
	Fatal("test fatal")

	expected := [...]string{"trace", "debug", "info", "warn", "error", "fatal"}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")

	if len(lines) != len(expected) {
		t.Errorf("Expected %d log entries, got %d\n", len(lines), len(expected))
		return
	}

	for i, l := range lines {
		line := strings.TrimSpace(l)

		e := expected[i]
		prefix := fmt.Sprintf("[%s] test %s", e, e)

		if !strings.HasPrefix(line, prefix) {
			t.Errorf("Expected prefix \"%s\" for line \"%s\"", prefix, line)
		}
	}

	// Close()
}
