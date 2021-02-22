package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/fatih/color"
)

// Transporter ist das Interface, das alle Funktionen für einen einzelnen Log-Transporter enthält.
type Transporter interface {
	minLevel() int

	transport(level Level, msg string, date time.Time)
	close()
}

// ConsoleTransporter ist der Transporter, der in die Konsole loggt.
// Hierbei existieren folgende Attribute:
//  Date: Datum soll mit ausgegeben werden
//  Colors: Eintrag soll farbig ausgegeben werden
//  MinLevel: nur Eintrage ab diesem Level sollen ausgegeben werden
type ConsoleTransporter struct {
	Date     bool
	Colors   bool
	MinLevel string

	lastMessage int64
}

func (t *ConsoleTransporter) minLevel() int {
	return Level(t.MinLevel).int()
}

func (t *ConsoleTransporter) transport(level Level, msg string, date time.Time) {
	const prefixLength = 5 + 2

	prefix := padStart("["+string(level)+"]", prefixLength, " ")

	if t.Colors {
		prefix = level.color(prefix)
	}

	var result bytes.Buffer
	result.WriteString(prefix)
	result.WriteRune(' ')

	if t.Date {
		dateStr := formatDate(date)

		if t.Colors {
			dateStr = color.WhiteString(dateStr)
		}

		result.WriteRune('[')
		result.WriteString(dateStr)
		result.WriteString("] ")
	}

	result.WriteString(msg)

	if t.lastMessage != 0 {
		diff := now() - t.lastMessage
		timeDiff := formatDiff(diff)

		result.WriteRune(' ')

		if t.Colors {
			timeDiff = color.WhiteString(timeDiff)
		}

		result.WriteString(timeDiff)
	}

	result.WriteRune('\n')

	t.lastMessage = now()
	os.Stdout.Write(result.Bytes())
}

func (t *ConsoleTransporter) close() {}

// ServerTransporter ist der Transporter, der an den Log-Server loggt.
// Hierbei existieren folgende Attribute:
//  Type: Typ, des Log-Clients. Nach gruppiert der Log-Server Log-Einträge.
//  URL: URL des Log-Servers
//  Secret: geheimer Token für den Log-Server
//  MinLevel: nur Eintrage ab diesem Level sollen ausgegeben werden
type ServerTransporter struct {
	Type   string
	URL    string
	Secret string

	MinLevel string

	queue          *queue
	lastErrorShown int64
}

type logEntry struct {
	Type    string `json:"type"`
	Level   Level  `json:"level"`
	Date    string `json:"date"`
	Message string `json:"message"`
	Secret  string `json:"secret,omitempty"`
}

type logError struct {
	Err string `json:"error"`
}

func (t *ServerTransporter) minLevel() int {
	return Level(t.MinLevel).int()
}

func (t *ServerTransporter) transport(level Level, msg string, date time.Time) {
	if t.queue == nil {
		t.runQueue()
	}

	e := logEntry{
		Type:    t.Type,
		Level:   level,
		Date:    date.Format(time.RFC3339),
		Message: msg,
	}

	if t.Secret != "" {
		e.Secret = t.Secret
	}

	t.queue.pushJob(e)
}

func (t *ServerTransporter) runQueue() {
	q := newQueue(func(v interface{}) {
		entry, ok := v.(logEntry)
		if !ok {
			return
		}

		client := http.Client{
			Timeout: time.Second * 10,
		}

		jsonData, err := json.Marshal(entry)
		buff := bytes.NewBuffer(jsonData)

		req, err := http.NewRequest(http.MethodPost, t.URL, buff)

		req.Header.Set("accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			t.showError(err)
			return
		}

		if res.Body != nil {
			defer res.Body.Close()
		}

		if res.StatusCode < 400 {
			return
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.showError(err)
			return
		}

		var logErr logError
		err = json.Unmarshal(body, &logErr)
		if err != nil {
			t.showError(err)
			return
		}

		if logErr.Err != "" {
			t.showError(errors.New(logErr.Err))
			return
		}
	}, 1)

	t.queue = q
}

func (t *ServerTransporter) showError(err error) {
	if t.lastErrorShown+10*int64(time.Minute) < now() {
		log := ConsoleTransporter{
			Colors: true,
		}

		date := time.Now()
		log.transport(levelError, "Failed to send log to server: "+err.Error(), date)

		t.lastErrorShown = now()
	}
}

func (t *ServerTransporter) close() {
	if t.queue != nil {
		t.queue.wait()
	}
}
