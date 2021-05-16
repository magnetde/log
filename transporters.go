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

	RotateBytes int64
	RotateLines int
	Rotations   int

	SuppressErrors bool

	file   *os.File
	fsize  int64
	flines int

	queue   *queue
	lastMsg int64
}

type fileLogEntry struct {
	level   Level
	message string
	date    time.Time
}

func (t *FileTransporter) Init() error {
	if t.MinLevel != "" && Level(t.MinLevel).Index() == 0 {
		t.MinLevel = ""
	}

	var err error
	t.file, err = os.OpenFile(t.Path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	stats, err := t.file.Stat()
	if err != nil {
		return err
	}

	t.fsize = stats.Size()

	t.flines, err = countLines(t.file)
	if err != nil {
		return err
	}

	t.runQueue()
	return nil
}

func (t *FileTransporter) runQueue() {
	q := newQueue(func(v interface{}) {
		e, ok := v.(fileLogEntry)
		if !ok {
			return
		}

		result := logToString(t, e.level, e.message, e.date)

		t.file.WriteString(result)
		t.fsize += int64(len([]byte(result)))
		t.flines += 1

		// Check if rotation needed
		if t.RotateBytes > 0 && t.fsize >= t.RotateBytes {
			t.rotate()
		} else if t.RotateLines > 0 && t.flines >= t.RotateLines {
			t.rotate()
		}
	}, 1, 1024)

	t.queue = q
}

var regexName = regexp.MustCompile(`(.+).(\d+).gz`)

func (t *FileTransporter) rotate() {
	if t.fsize == 0 || t.flines == 0 {
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

func (t *FileTransporter) showError(err error) {
	if !t.SuppressErrors {
		log := ConsoleTransporter{Colors: true}
		date := time.Now()
		log.Transport(levelError, "Failed to rotate log file: "+err.Error(), date)
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
	if !level.GreaterEquals(Level(t.MinLevel)) {
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

	SuppressErrors bool

	queue          *queue
	lastErrorShown int64
}

type serverLogEntry struct {
	Type    string `json:"type"`
	Level   Level  `json:"level"`
	Date    string `json:"date"`
	Message string `json:"message"`
	Secret  string `json:"secret,omitempty"`
}

type logError struct {
	Err string `json:"error"`
}

func (t *ServerTransporter) Init() error {
	if t.MinLevel != "" && Level(t.MinLevel).Index() == 0 {
		t.MinLevel = ""
	}

	t.runQueue()
	return nil
}

func (t *ServerTransporter) runQueue() {
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
	}, 1, 1024)

	t.queue = q
}

func (t *ServerTransporter) showError(err error) {
	if !t.SuppressErrors && t.lastErrorShown+10*int64(time.Minute) < now() {
		log := ConsoleTransporter{Colors: true}
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

	e := serverLogEntry{
		Type:    t.Type,
		Level:   level,
		Date:    date.Format(time.RFC3339),
		Message: msg,
	}

	if t.Secret != "" {
		e.Secret = t.Secret
	}

	t.queue.addJob(e)
}

// Close waits until the log entries have been sent to the server and then deletes the queue.
func (t *ServerTransporter) Close() {
	t.queue.close()
}
