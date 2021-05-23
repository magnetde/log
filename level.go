package log

import "github.com/fatih/color"

// Level is the internal log level.
type Level string

const (
	levelTrace Level = "trace"
	levelDebug Level = "debug"
	levelInfo  Level = "info"
	levelWarn  Level = "warn"
	levelError Level = "error"
	levelFatal Level = "fatal"
)

func init() {
	color.NoColor = false
}

// Index returns the severity of the level.
func (l Level) Index() int {
	switch l {
	case levelTrace:
		return 1
	case levelDebug:
		return 2
	case levelInfo:
		return 3
	case levelWarn:
		return 4
	case levelError:
		return 5
	case levelFatal:
		return 6
	default:
		return 0
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
	return colors[l.Index()].Sprint(str)
}
