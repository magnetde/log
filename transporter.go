package log

import (
	"strings"
	"time"

	"github.com/fatih/color"
)

// Transporter is the interface that contains all the functions for a single log transporter.
type Transporter interface {
	Transport(level Level, msg string, date time.Time)
}

// initTransporter is the transporter with an init function.
type initTransporter interface {
	Transporter
	Init() error
}

// closeTransporter is the transporter with an close function.
type closeTransporter interface {
	Transporter
	Close()
}

type stringTransporter interface {
	withDate() bool
	withColors() bool

	lastMessage() int64
	setLastMessage(l int64)
}

// 5 (length of trace, debug, error) + 2 (length auf two brackets)
const prefixLength = 5 + 2

func logToString(t stringTransporter, level Level, msg string, date time.Time) string {
	prefix := padStart("["+string(level)+"]", prefixLength, " ")

	if t.withColors() {
		prefix = level.color(prefix)
	} else {
		msg = removeColors(msg)
	}

	var result strings.Builder
	result.WriteString(prefix)

	if t.withDate() {
		dateStr := formatDate(date)

		if t.withColors() {
			dateStr = color.WhiteString(dateStr)
		}

		result.WriteString(" [")
		result.WriteString(dateStr)
		result.WriteString("]")
	}

	if len(msg) > 0 {
		result.WriteRune(' ')
		result.WriteString(msg)
	}

	if t.lastMessage() != 0 {
		diff := now() - t.lastMessage()
		timeDiff := formatDiff(diff)

		if t.withColors() {
			timeDiff = color.WhiteString(timeDiff)
		}

		result.WriteRune(' ')
		result.WriteString(timeDiff)
	}

	result.WriteRune('\n')

	t.setLastMessage(now())
	return result.String()
}
