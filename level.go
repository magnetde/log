package log

import "github.com/fatih/color"

// Level ist das interne Log-Level.
type Level string

const (
	levelTrace Level = "trace"
	levelDebug Level = "debug"
	levelInfo  Level = "info"
	levelWarn  Level = "warn"
	levelError Level = "error"
	levelFatal Level = "fatal"
)

func (l Level) int() int {
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
