package log

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"
)

const regexLevel = "trace|debug|info|warn|error|fatal"
const regexDate = "[0-9]+-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}"

var regexLog = regexp.MustCompile(`^\[(` + regexLevel + `)\]( \[(` + regexDate + `)\])? ?(.*)$`)
var regexTime = regexp.MustCompile(` (0|0\.[0-9][0-9][0-9]? ms|[0-9]+ (ms|s|m|h))$`)

type ParsedLog struct {
	level    string
	date     string
	message  string
	timediff string
}

func parseLog(line string) *ParsedLog {
	line = strings.TrimSpace(line)

	groups := regexLog.FindStringSubmatch(line)

	parsed := &ParsedLog{
		level: groups[1],
		date:  groups[3],
	}

	timeGroups := regexTime.FindStringSubmatch(groups[4])

	if len(timeGroups) > 0 {
		timediff := timeGroups[1]

		parsed.message = strings.TrimSuffix(groups[4], " "+timediff)
		parsed.timediff = timediff
	} else {
		parsed.message = groups[4]
	}

	return parsed
}

// Testing with default transport: console transport with output = nil
func TestDefault(t *testing.T) {
	Info("test")
}

func TestBasic(t *testing.T) {
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
		t.Fatalf("Expected %d log entries, got %d\n", len(lines), len(expected))
		return
	}

	for i, l := range lines {
		line := strings.TrimSpace(l)

		parsed := parseLog(line)
		e := expected[i]

		if parsed.level != e {
			t.Errorf("Expected level \"%s\", got \"%s\"", e, parsed.level)
		}

		if parsed.message != "test "+e {
			t.Errorf("Expected message \"test %s\", got \"%s\"", e, parsed.message)
		}
	}

	// Only for coverage
	Close()
}

func TestDate(t *testing.T) {
	var buf bytes.Buffer

	Init(&ConsoleTransporter{
		Colors: false,
		Date:   true,
		Output: &buf,
	})

	Info("test date")

	msg := strings.TrimSpace(buf.String())

	parsed := parseLog(msg)

	layout := strings.Replace(time.RFC3339, "T", " ", 1)
	layout = strings.Split(layout, "Z")[0]

	logTime, err := time.Parse(layout, parsed.date)
	if err != nil {
		t.Fatalf("Failed to parse log date: %s", err.Error())
	}

	now, _ := time.Parse(layout, formatDate(time.Now()))
	diff := now.Sub(logTime)

	if diff < 0 {
		t.Fatalf("The log time is in the future: %d", diff)
	}

	if diff >= 1*time.Minute {
		t.Fatalf("The log entry was created more than a minute ago")
	}
}
