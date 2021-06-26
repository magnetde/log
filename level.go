package serverhook

import (
	"encoding/json"
	"fmt"
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
	LevelPanic
)

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
	case LevelPanic:
		return "panic"
	default:
		return ""
	}
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
	case "panic":
		*l = LevelPanic
	default:
		return fmt.Errorf(`unknown level string "%s"`, s)
	}

	return nil
}
