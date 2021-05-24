package log

// File with different implementations of transporters.
// - ConsoleTransporter: log to console or another writer
// - FileTransporter: write logs to file and rotate log files
// - ServerTransporter: send logs to logcollect server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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
	if level.Index() < Level(t.MinLevel).Index() {
		return
	}

	if t.Output == nil {
		t.Output = os.Stdout
	}

	result := logToString(t, level, msg, date)
	t.Output.Write([]byte(result))
}

// FileTransporter writes log entries to a file.
type FileTransporter struct {
	Path string

	Date     bool
	Colors   bool
	MinLevel string

	RotateBytes int64
	RotateLines int
	Rotations   int

	SuppressErrors bool

	file   *os.File
	fsize  int64
	flines int

	closed  bool
	queue   *queue
	lastMsg int64
}

// fileLogEntry is used for elements on the queue
type fileLogEntry struct {
	level   Level
	message string
	date    time.Time
}

// Init opens the log file.
// If rotation is enabled, the file size or the number of lines in the file is also counted, if necessary.
func (t *FileTransporter) Init() error {
	if t.MinLevel != "" && Level(t.MinLevel).Index() == 0 {
		t.MinLevel = ""
	}

	var err error
	t.file, err = os.OpenFile(t.Path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	if t.RotateBytes > 0 {
		stats, err := t.file.Stat()
		if err != nil {
			return err
		}

		t.fsize = stats.Size()
	} else {
		t.fsize = 0
	}

	if t.RotateLines > 0 {
		t.flines, err = countLines(t.file)
		if err != nil {
			return err
		}
	} else {
		t.flines = 0
	}

	t.closed = false
	t.queue = t.runQueue()
	t.lastMsg = 0

	return nil
}

// runQueue creates the queue that runs jobs in the background.
func (t *FileTransporter) runQueue() *queue {
	q := newQueue(func(v interface{}) {
		e, ok := v.(fileLogEntry)
		if !ok {
			return
		}

		result := logToString(t, e.level, e.message, e.date)

		n, err := t.file.WriteString(result)
		if err != nil {
			t.showError(err)
			return
		}

		if t.RotateBytes > 0 {
			t.fsize += int64(n)
		}
		if t.RotateLines > 0 {
			t.flines++
		}

		// Check if rotation needed
		if t.RotateBytes > 0 && t.fsize >= t.RotateBytes {
			t.rotate()
		} else if t.RotateLines > 0 && t.flines >= t.RotateLines {
			t.rotate()
		}
	}, 1, 1024)

	return q
}

var regexName = regexp.MustCompile(`(.+).(\d+).gz`)

// rotate rotates the current log file by compressing it and renaming or deleting previous rotations.
func (t *FileTransporter) rotate() {
	if (t.RotateBytes > 0 && t.fsize == 0) || (t.RotateLines > 0 && t.flines == 0) {
		return
	}

	dir := filepath.Dir(t.Path)
	prefix := strings.TrimSpace(filepath.Base(t.Path))

	newArchive := filepath.Join(dir, prefix+".1.gz")

	// Rotate archives while xxx.1.gz exists
	for {
		exists, err := fileExists(newArchive)

		if exists && err == nil {
			err = t.rotateArchives(dir, prefix)
			if err != nil {
				t.showError(err)
				break
			}
		} else {
			if err != nil {
				t.showError(err)
			}

			break
		}
	}

	// Write bytes in compressed form to the file.
	gz, err := os.OpenFile(newArchive, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		t.showError(err)
		return
	}

	w := gzip.NewWriter(gz)
	defer w.Close()

	t.file.Seek(0, io.SeekStart)
	_, err = io.Copy(w, t.file)
	if err != nil {
		t.showError(err)
		return
	}

	err = t.file.Truncate(0)
	if err != nil {
		t.showError(err)
		return
	}

	t.fsize = 0
	t.flines = 0
}

// rotateArchives rotates by incrementing the counter of each rotation by one (example: log.3.gz -> log.4.gz)
func (t *FileTransporter) rotateArchives(dir string, prefix string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	renames := make(map[string]string)

	for _, file := range files {
		name := file.Name()
		path := filepath.Join(dir, name)

		groups := regexName.FindStringSubmatch(name)
		if len(groups) == 0 {
			continue
		}

		p := strings.TrimSpace(groups[1])
		if prefix != p {
			continue
		}

		index, err := strconv.Atoi(groups[2])
		if err != nil {
			continue
		}

		if t.Rotations > 0 && index+1 >= t.Rotations { // Rotate
			os.Remove(path)
			continue
		}

		newName := fmt.Sprintf("%s.%d.gz", prefix, index+1)
		newPath := filepath.Join(dir, newName)

		renames[path] = newPath
	}

	return renameAll(renames)
}

// showError prints an error to the console.
func (t *FileTransporter) showError(err error) {
	if !t.SuppressErrors {
		log := ConsoleTransporter{Colors: true}
		date := time.Now()
		log.Transport(levelError, "Failed to write log file: "+err.Error(), date)
	}
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
	if t.closed || level.Index() < Level(t.MinLevel).Index() {
		return
	}

	e := fileLogEntry{
		level:   level,
		message: msg,
		date:    date,
	}

	t.queue.addJob(e)
}

// Close closes the log file.
func (t *FileTransporter) Close() {
	t.closed = true
	t.queue.close()
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

	KeepColors     bool
	SuppressErrors bool

	closed         bool
	queue          *queue
	lastErrorShown int64
}

// serverLogEntry is used to serialize JSON.
type serverLogEntry struct {
	Type    string    `json:"type"`
	Level   Level     `json:"level"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
	Secret  string    `json:"secret,omitempty"`
}

type logError struct {
	Err string `json:"error"`
}

// Init initializes the logger by starting the queue among other things.
func (t *ServerTransporter) Init() error {
	if t.Type == "" {
		return errors.New("empty log type")
	}
	if t.URL == "" {
		return errors.New("empty url")
	}
	if t.MinLevel != "" && Level(t.MinLevel).Index() == 0 {
		t.MinLevel = ""
	}

	t.closed = false
	t.queue = t.runQueue()
	t.lastErrorShown = 0

	return nil
}

// runQueue creates the queue that runs jobs in the background.
func (t *ServerTransporter) runQueue() *queue {
	q := newQueue(func(v interface{}) {
		entry, ok := v.(serverLogEntry)
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

		r := bytes.NewReader(jsonData)

		req, err := http.NewRequest(http.MethodPost, t.URL, r)
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
	}, 1, 1024)

	return q
}

// showError prints an error to the console.
func (t *ServerTransporter) showError(err error) {
	if !t.SuppressErrors && t.lastErrorShown+10*int64(time.Minute) < now() {
		log := ConsoleTransporter{Colors: true}
		date := time.Now()
		log.Transport(levelError, "Failed to send log to server: "+err.Error(), date)

		t.lastErrorShown = now()
	}
}

// Transport sends the log entry to the server.
func (t *ServerTransporter) Transport(level Level, msg string, date time.Time) {
	if t.closed || level.Index() < Level(t.MinLevel).Index() {
		return
	}

	if !t.KeepColors {
		msg = removeColors(msg)
	}

	e := serverLogEntry{
		Type:    t.Type,
		Level:   level,
		Date:    date,
		Message: msg,
	}

	if t.Secret != "" {
		e.Secret = t.Secret
	}

	t.queue.addJob(e)
}

// Close waits until the log entries have been sent to the server and then deletes the queue.
func (t *ServerTransporter) Close() {
	t.closed = true
	t.queue.close()
}
