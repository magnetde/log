package log

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

var l *Logger

func init() {
	ts := &ConsoleTransporter{
		Colors: true,
	}

	l, _ = CreateLogger(ts)
}

// Logger is a data structure that can be used to log.
// Usually it is used by the global logger. However, different loggers can also be created.
type Logger struct {
	mu *sync.Mutex
	ts []Transporter
}

// CreateLogger creates a new logger data structure.
// In addition to the console transporter, logging can also be send to a server.
// If an transporter implements the Init-function and it returns an error, the error is returned
// and the logger will not be initialiazed.
func CreateLogger(ts ...Transporter) (*Logger, error) {
	l := &Logger{
		mu: new(sync.Mutex),
	}

	err := l.init(ts...)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// init initializes the logger by adding the given transporters to the logger.
func (l *Logger) init(ts ...Transporter) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, t := range ts {
		if it, ok := t.(initTransporter); ok {
			err := it.Init()
			if err != nil {
				return err
			}
		}
	}

	l.ts = ts
	return nil
}

// Log performs the respective logging by sending the log entry to all transporters.
func (l *Logger) Log(level Level, a ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	date := time.Now()

	var buff bytes.Buffer

	length := len(a)
	for i, v := range a {
		buff.WriteString(fmt.Sprintf("%+v", v))
		if i < length-1 {
			buff.WriteRune(' ')
		}
	}

	msg := buff.String()

	for _, t := range l.ts {
		t.Transport(level, msg, date)
	}
}

// Close closes all transporters of the logger.
// When logging is sent to a server, the function waits until all log entries have been successfully sent to the server.
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, transport := range l.ts {
		if ct, ok := transport.(closeTransporter); ok {
			ct.Close()
		}
	}

	l.ts = []Transporter{}
}

// Init initializes the global logger by adding the given transporters to the logger.
// This call is optional. If this function is not called, it will only be logged to the console.
//
// Warning: This call does not call the close function, which is used to close a logger again.
// If a logger still needs to be closed (e.g. FileLogger) it must be closed first.
func Init(ts ...Transporter) error {
	return l.init(ts...)
}

// Trace creates a log entry with the "trace" level
func Trace(a ...interface{}) {
	l.Log(levelTrace, a...)
}

// Debug creates a log entry with the "debug" level
func Debug(a ...interface{}) {
	l.Log(levelDebug, a...)
}

// Info creates a log entry with the "info" level
func Info(a ...interface{}) {
	l.Log(levelInfo, a...)
}

// Warn creates a log entry with the "warn" level
func Warn(a ...interface{}) {
	l.Log(levelWarn, a...)
}

// Error creates a log entry with the "error" level
func Error(a ...interface{}) {
	l.Log(levelError, a...)
}

// Fatal creates a log entry with the "fatal" level
func Fatal(a ...interface{}) {
	l.Log(levelFatal, a...)
}

// Close closes all transporters.
// When logging is sent to a server, the function waits until all log entries have been successfully sent to the server.
func Close() {
	l.Close()
}
