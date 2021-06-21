package serverhook

// Option is the parameter type for options when initializing the log hook.
type Option interface {
	apply(h *ServerHook)
}

// WithSecret - secret needed for logcollect server
func WithSecret(secret string) Option {
	return secretOption(secret)
}

type secretOption string

func (o secretOption) apply(h *ServerHook) {
	h.secret = string(o)
}

// KeepColors - keep ANSII colors before sending them to the log server.
func KeepColors(val bool) Option {
	return keepColorOption(val)
}

type keepColorOption bool

func (o keepColorOption) apply(h *ServerHook) {
	h.keepColors = bool(o)
}

// SuppressErrors - suppress send errors.
func SuppressErrors(val bool) Option {
	return suppressErrorOption(val)
}

type suppressErrorOption bool

func (o suppressErrorOption) apply(h *ServerHook) {
	h.suppressErrors = bool(o)
}

// Synchronous - send log entries synchronous to the server.
func Synchronous(val bool) Option {
	return synchronousOption(val)
}

type synchronousOption bool

func (o synchronousOption) apply(h *ServerHook) {
	h.synchronous = bool(o)
}
