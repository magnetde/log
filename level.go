package log

import "github.com/fatih/color"

// Level is the internal log level.
type Level int

const (
	LevelTrace Level = iota + 1
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func init() {
	color.NoColor = false
}

func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "trace"
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	default:
		return ""
	}
}

var colors = []*color.Color{
	color.New(),
	color.New(color.FgBlue),
	color.New(color.FgCyan),
	color.New(color.FgGreen),
	color.New(color.FgYellow),
	color.New(color.FgRed),
	color.New(color.FgRed, color.Bold),
}

// color changes the color of a string to the color assigned to the level.
func (l Level) color(str string) string {
	return colors[l].Sprint(str)
}
