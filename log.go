package log

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

var transports []Transporter
var mutex *sync.Mutex

func init() {
	transport := ConsoleTransporter{
		Colors: true,
	}

	transports = append(transports, &transport)
	mutex = new(sync.Mutex)
}

// Init initializes the logger by adding the given transporters to the logger.
// In addition to the console transporter, logging can also be send to a server.
//
// This call is optional. If this function is not called, it will only be logged to the console.
func Init(t ...Transporter) {
	mutex.Lock()
	defer mutex.Unlock()

	transports = t
}

// Close closes all transporters.
// When logging is sent to a server, the function waits until all log entries have been successfully sent to the server.
func Close() {
	mutex.Lock()
	defer mutex.Unlock()

	for _, transport := range transports {
		transport.Close()
	}
	transports = []Transporter{}
}

func logInternal(level Level, a []interface{}) {
	mutex.Lock()
	defer mutex.Unlock()

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

	for _, t := range transports {
		t.Transport(level, msg, date)
	}
}

// Trace creates a log entry with the "trace" level
func Trace(a ...interface{}) {
	logInternal(levelTrace, a)
}

// Debug creates a log entry with the "debug" level
func Debug(a ...interface{}) {
	logInternal(levelDebug, a)
}

// Info creates a log entry with the "info" level
func Info(a ...interface{}) {
	logInternal(levelInfo, a)
}

// Warn creates a log entry with the "warn" level
func Warn(a ...interface{}) {
	logInternal(levelWarn, a)
}

// Error creates a log entry with the "error" level
func Error(a ...interface{}) {
	logInternal(levelError, a)
}

// Fatal creates a log entry with the "fatal" level
func Fatal(a ...interface{}) {
	logInternal(levelFatal, a)
}
