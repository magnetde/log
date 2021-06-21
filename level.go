package serverhook

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
)

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

func (l Level) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

func (l *Level) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	switch s {
	case "trace":
		*l = LevelTrace
	case "debug":
		*l = LevelDebug
	case "info":
		*l = LevelInfo
	case "warn":
		*l = LevelWarn
	case "error":
		*l = LevelError
	case "fatal":
		*l = LevelFatal
	default:
		return fmt.Errorf(`unknown level string "%s"`, s)
	}

	return nil
}
