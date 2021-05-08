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
	if len(groups) == 0 {
		return nil
	}

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
	Info()
}

func TestLevels(t *testing.T) {
	var buf bytes.Buffer

	Init(&ConsoleTransporter{
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
		if parsed == nil {
			t.Errorf("Filed to parse log entry \"%s\"", line)
			continue
		}

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
		Date:   true,
		Output: &buf,
	})

	Info("test date")

	msg := strings.TrimSpace(buf.String())

	parsed := parseLog(msg)
	if parsed == nil {
		t.Fatalf("Filed to parse log entry \"%s\"", msg)
	}

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

func TestMinLevel(t *testing.T) {
	var buf bytes.Buffer

	Init(&ConsoleTransporter{
		MinLevel: "warn",
		Output:   &buf,
	})

	Trace("test")
	Debug("test")
	Info("test")
	Warn("test")
	Error("test")
	Fatal("test")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")

	for _, l := range lines {
		line := strings.TrimSpace(l)

		parsed := parseLog(line)
		if parsed == nil {
			t.Errorf("Filed to parse log entry \"%s\"", line)
			continue
		}

		if parsed.level == "trace" || parsed.level == "debug" || parsed.level == "info" {
			t.Errorf("Log entry with level %s found", parsed.level)
		}
	}
}

func TestConcat(t *testing.T) {
	var buf bytes.Buffer

	Init(&ConsoleTransporter{
		Output: &buf,
	})

	Info("abc", 1, -1, 0.5, true, nil)

	msg := buf.String()
	parsed := parseLog(msg)
	if parsed == nil {
		t.Errorf("Filed to parse log entry \"%s\"", msg)
	} else if parsed.message != "abc 1 -1 0.5 true <nil>" {
		t.Errorf("Concating and converting values to string does not work")
	}
}

func TestColor(t *testing.T) {
	var buf bytes.Buffer

	Init(&ConsoleTransporter{
		Colors: true,
		Date:   true,
		Output: &buf,
	})

	Trace("test")
	Debug("test")
	Info("test")
	Warn("test")
	Error("test")
	Fatal("test")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")

	regexLevel := regexp.MustCompile(`\[([a-z]+)\]`)

	for i, line := range lines {
		levels := regexLevel.FindStringSubmatch(line)
		if len(levels) == 0 {
			t.Errorf("Failed to find the level at the colored log entry at entry %d", i)
			continue
		}

		level := levels[1]

		prefix := []byte{27, '['} // Prefix ^[

		switch level {
		case "trace":
			prefix = append(prefix, []byte("34m")...) // blue
		case "debug":
			prefix = append(prefix, []byte("36m")...) // cyan
		case "info":
			prefix = append(prefix, []byte("32m")...) // green
		case "warn":
			prefix = append(prefix, []byte("33m")...) // yellow
		case "error":
			prefix = append(prefix, []byte("31m")...) // red
		case "fatal":
			prefix = append(prefix, []byte("31;1m")...) // red + bold
		default:
			t.Errorf("Unknown log level %s", level)
			continue
		}

		if !bytes.HasPrefix([]byte(line), prefix) {
			t.Errorf("Wrong color at log level %s", level)
			continue
		}
	}
}
