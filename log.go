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

// Init initialisiert den Logger, indem verschiedene Transporter zu dem Logger hinzugefügt werden.
// Hierbei kann neben dem Transporter in die Konsole auch an einem Server geloggt werden.
//
// Dieser Aufruf ist optional. Wenn diese Funktion nicht aufgerufen wird, wird nur in die Konsole geloggt.
func Init(t ...Transporter) {
	mutex.Lock()
	defer mutex.Unlock()

	transports = nil

	for _, transport := range t {
		transports = append(transports, transport)
	}
}

// Close schließt alle Transporter.
// Wenn an einen Server geloggt wird, wird gewartet, bis alle Log-Einträge erfolgreich an den Server gesendet wurden.
func Close() {
	mutex.Lock()
	defer mutex.Unlock()

	for _, transport := range transports {
		transport.close()
	}
	transports = []Transporter{}
}

func logInternal(level Level, a ...interface{}) {
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
		if level.int() < t.minLevel() {
			return
		}

		t.transport(level, msg, date)
	}
}

// Trace erstellt einen Log-Eintrag mit dem Level "trace"
func Trace(a ...interface{}) {
	logInternal(levelTrace, a...)
}

// Debug erstellt einen Log-Eintrag mit dem Level "debug"
func Debug(a ...interface{}) {
	logInternal(levelDebug, a...)
}

// Info erstellt einen Log-Eintrag mit dem Level "Info"
func Info(a ...interface{}) {
	logInternal(levelInfo, a...)
}

// Warn erstellt einen Log-Eintrag mit dem Level "warn"
func Warn(a ...interface{}) {
	logInternal(levelWarn, a...)
}

// Error erstellt einen Log-Eintrag mit dem Level "error"
func Error(a ...interface{}) {
	logInternal(levelError, a...)
}

// Fatal erstellt einen Log-Eintrag mit dem Level "fatal"
func Fatal(a ...interface{}) {
	logInternal(levelFatal, a...)
}
