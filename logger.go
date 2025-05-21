package timewheel

import "log"

// Logger is logger interface.
type Logger interface {
	Printf(string, ...any)
}

// LoggerFunc is a bridge between Logger and any third party logger.
type LoggerFunc func(string, ...any)

// Printf implements Logger interface.
func (f LoggerFunc) Printf(msg string, args ...any) { f(msg, args...) }

// defaultLogger writes nothing.
var defaultLogger = LoggerFunc(func(string, ...any) {})

// Printf is a logger which wraps log.Printf
var Printf = LoggerFunc(log.Printf)
