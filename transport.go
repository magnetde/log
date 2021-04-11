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

// Transporter is the interface that contains all the functions for a single log transporter.
type Transporter interface {
	Transport(level Level, msg string, date time.Time)
	Close()
}

// ConsoleTransporter is the transporter that logs to the console.
// The following attributes exist:
//  Date: the date should be included in the output
//  Colors: output should be colored
//  MinLevel: only entries with a log level greater than or equal to this level should be printed
type ConsoleTransporter struct {
	Date     bool
	Colors   bool
	MinLevel string

	lastMessage int64
}

// Transport prints the log entry.
func (t *ConsoleTransporter) Transport(level Level, msg string, date time.Time) {
	if !level.GreaterEquals(Level(t.MinLevel)) {
		return
	}

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

// Close does nothing. Its only purpose is to match the Transporter interface.
func (t *ConsoleTransporter) Close() {}

// ServerTransporter is the transporter that logs to the log server.
// The following attributes exists:
//  Type: type of the log client. The log server groups log entries according to this param
//  URL: URL of the log server
//  Secret: secret token for the log server
//  MinLevel: only entries from this level should be sent
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

// Transport send the log entry to the server.
func (t *ServerTransporter) Transport(level Level, msg string, date time.Time) {
	if !level.GreaterEquals(Level(t.MinLevel)) {
		return
	}

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
		log.Transport(levelError, "Failed to send log to server: "+err.Error(), date)

		t.lastErrorShown = now()
	}
}

// Close waits until the log entries have been sent to the server and then deletes the queue.
func (t *ServerTransporter) Close() {
	if t.queue != nil {
		t.queue.stopQueue()
		t.queue.wait()
		t.queue = nil
	}
}
