package serverhook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// BufSize is used as the channel size which buffers log entries before sending them asynchrously to the log server.
// Set serverlog.BufSize = <value> _before_ calling NewServerHook
// Once the buffer is full, logging will start blocking, waiting for slots to be available in the queue.
var BufSize uint = 8192

// ServerHook to send logs to logcollect server.
type ServerHook struct {
	typ string
	url string

	secret         string
	keepColors     bool
	suppressErrors bool

	synchronous bool
	buf         chan *logrus.Entry
	wg          sync.WaitGroup
	mu          sync.RWMutex

	nextError time.Time
}

// Test if the ServerHook matches the logrus.Hook interface.
var _ logrus.Hook = (*ServerHook)(nil)

// NewServerHook creates a hook to be added to an instance of logger.
func NewServerHook(typ, url string, options ...Option) (*ServerHook, error) {
	if typ == "" {
		return nil, errors.New("empty log type")
	}
	if url == "" {
		return nil, errors.New("empty url")
	}

	h := &ServerHook{
		typ: typ,
		url: url,
	}

	for _, o := range options {
		o.apply(h)
	}

	if !h.synchronous {
		h.buf = make(chan *logrus.Entry, BufSize)

		go h.worker()
	}

	return h, nil
}

// Fire sends a log entry to the server.
func (h *ServerHook) Fire(entry *logrus.Entry) error {
	h.mu.RLock() // Claim the mutex as a RLock - allowing multiple go routines to log simultaneously
	defer h.mu.RUnlock()

	// Creating a new entry to prevent data races
	newData := make(map[string]interface{})
	for k, v := range entry.Data {
		newData[k] = v
	}

	newEntry := &logrus.Entry{
		Logger:  entry.Logger,
		Data:    newData,
		Time:    entry.Time,
		Level:   entry.Level,
		Caller:  entry.Caller,
		Message: entry.Message,
	}

	if h.synchronous {
		h.sendEntry(newEntry)
	} else {
		h.wg.Add(1)
		h.buf <- newEntry
	}

	if entry.Level == logrus.PanicLevel || entry.Level == logrus.FatalLevel {
		h.wg.Wait()
	}

	return nil
}

// Flush waits for the log queue to be empty.
// This func is meant to be used when the hook was created as asynchronous.
func (h *ServerHook) Flush() {
	h.mu.Lock() // claim the mutex as a Lock - we want exclusive access to it
	defer h.mu.Unlock()

	h.wg.Wait()
}

// Levels returns the Levels used for this hook.
func (h *ServerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// process runs the worker queue in the background
func (h *ServerHook) worker() {
	for {
		entry := <-h.buf // receive new entry on channel
		h.sendEntry(entry)
		h.wg.Done()
	}
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

func (h *ServerHook) sendEntry(entry *logrus.Entry) {
	e := h.createServerEntry(entry)

	jsonData, err := json.Marshal(e)
	if err != nil {
		h.showError(err)
		return
	}

	r := bytes.NewReader(jsonData)

	req, err := http.NewRequest(http.MethodPost, h.url, r)
	if err != nil {
		h.showError(err)
		return
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{
		Timeout: time.Second * 10,
	}

	res, err := client.Do(req)
	if err != nil {
		h.showError(err)
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
		h.showError(err)
		return
	}

	var logErr logError
	err = json.Unmarshal(body, &logErr)
	if err != nil {
		h.showError(err)
		return
	}

	if logErr.Err != "" {
		h.showError(errors.New(logErr.Err))
		return
	}
}

// showError prints an error to the console.
func (h *ServerHook) showError(err error) {
	if !h.suppressErrors && h.nextError.Before(time.Now()) {
		logrus.Error("Failed to send log to server: " + err.Error())

		h.nextError = time.Now().Add(10 * time.Minute)
	}
}

// createServerEntry creates a log entry which can be send to the log server from a logrus entry.
func (h *ServerHook) createServerEntry(entry *logrus.Entry) *serverLogEntry {
	var lvl Level
	switch entry.Level {
	case logrus.PanicLevel, logrus.FatalLevel:
		lvl = LevelFatal
	case logrus.ErrorLevel:
		lvl = LevelError
	case logrus.WarnLevel:
		lvl = LevelWarn
	case logrus.InfoLevel:
		lvl = LevelInfo
	case logrus.DebugLevel:
		lvl = LevelDebug
	case logrus.TraceLevel:
		lvl = LevelTrace
	default: // should never happen
		break
	}

	var b strings.Builder
	b.WriteString(entry.Message)
	appendData(&b, entry.Data)

	msg := b.String()
	if !h.keepColors {
		msg = removeColors(msg)
	}

	e := &serverLogEntry{
		Type:    h.typ,
		Level:   lvl,
		Date:    entry.Time,
		Message: msg,
		Secret:  h.secret,
	}

	return e
}

// appendData appends the data to the log message.
func appendData(b *strings.Builder, data logrus.Fields) {
	keys := make([]string, 0, len(data))

	for k := range data {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		if v, ok := data[k]; ok {
			b.WriteRune(' ')
			appendKeyValue(b, k, v)
		}
	}
}

// appendKeyValue appends the key and the value to the log message.
func appendKeyValue(b *strings.Builder, key string, value interface{}) {
	b.WriteString(key)
	b.WriteByte('=')

	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	b.WriteString(quoteIfNeeded(stringVal))
}
