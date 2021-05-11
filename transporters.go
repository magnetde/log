package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// ConsoleTransporter is the transporter that logs to the console.
// The following attributes exist:
//  Date: the date should be included in the output
//  Colors: output should be colored
//  MinLevel: only entries with a log level greater than or equal to this level should be printed
type ConsoleTransporter struct {
	Date     bool
	Colors   bool
	MinLevel string
	Output   io.Writer

	lastMsg int64
}

func (t *ConsoleTransporter) withDate() bool {
	return t.Date
}

func (t *ConsoleTransporter) withColors() bool {
	return t.Colors
}

func (t *ConsoleTransporter) lastMessage() int64 {
	return t.lastMsg
}

func (t *ConsoleTransporter) setLastMessage(l int64) {
	t.lastMsg = l
}

// Transport prints the log entry.
func (t *ConsoleTransporter) Transport(level Level, msg string, date time.Time) {
	if !level.GreaterEquals(Level(t.MinLevel)) {
		return
	}

	if t.Output == nil {
		t.Output = os.Stdout
	}

	result := logToString(t, level, msg, date)
	t.Output.Write([]byte(result))
}

type FileTransporter struct {
	Path string

	Date     bool
	Colors   bool
	MinLevel string

	file    *os.File
	lastMsg int64
}

func (t *FileTransporter) Init() error {
	var err error
	t.file, err = os.OpenFile(t.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	return err
}

func (t *FileTransporter) withDate() bool {
	return t.Date
}

func (t *FileTransporter) withColors() bool {
	return t.Colors
}

func (t *FileTransporter) lastMessage() int64 {
	return t.lastMsg
}

func (t *FileTransporter) setLastMessage(l int64) {
	t.lastMsg = l
}

// Transport writes the log entry to the file.
func (t *FileTransporter) Transport(level Level, msg string, date time.Time) {
	if !level.GreaterEquals(Level(t.MinLevel)) {
		return
	}

	result := logToString(t, level, msg, date)
	t.file.WriteString(result)
}

// Close closes the log file.
func (t *FileTransporter) Close() {
	t.file.Close()
}

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

func (t *ServerTransporter) Init() {
	if t.MinLevel != "" && Level(t.MinLevel).num() == 0 {
		t.MinLevel = ""
	}

	t.runQueue()
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
		if err != nil {
			t.showError(err)
			return
		}

		buff := bytes.NewBuffer(jsonData)

		req, err := http.NewRequest(http.MethodPost, t.URL, buff)
		if err != nil {
			t.showError(err)
			return
		}

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

// Transport send the log entry to the server.
func (t *ServerTransporter) Transport(level Level, msg string, date time.Time) {
	if !level.GreaterEquals(Level(t.MinLevel)) {
		return
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

// Close waits until the log entries have been sent to the server and then deletes the queue.
func (t *ServerTransporter) Close() {
	if t.queue != nil {
		t.queue.stopQueue()
		t.queue.wait()
		t.queue = nil
	}
}
