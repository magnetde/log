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

// GreaterEquals compares whether the current level is greater than or equal to the given minimum level.
func (l Level) GreaterEquals(min Level) bool {
	return l.Index() >= min.Index()
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

// color colors the givewn text in the color assigned to the level.
func (l Level) color(str string) string {
	switch l {
	case levelTrace:
		return color.BlueString(str)
	case levelDebug:
		return color.CyanString(str)
	case levelInfo:
		return color.GreenString(str)
	case levelWarn:
		return color.YellowString(str)
	case levelError:
		return color.RedString(str)
	case levelFatal:
		boldRed := color.New(color.FgRed, color.Bold)
		return boldRed.Sprint(str)
	default:
		return ""
	}
}
