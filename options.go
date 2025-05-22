package timewheel

import (
	"time"
)

// default is a 1 minute time wheel with 60 slots and 1 second tick duration.
const (
	defaultSlotNum      = 60
	defaultTickDuration = 1 * time.Second
)

// Options is common options
type Options struct {
	Handler      Handler
	Logger       Logger
	SlotNum      int
	TickDuration time.Duration
}

// NewOptions creates options with defaults.
func NewOptions(opts ...Option) Options {
	var options = Options{
		Handler:      defaultHandler,
		Logger:       defaultLogger,
		SlotNum:      defaultSlotNum,
		TickDuration: defaultTickDuration,
	}
	for _, opt := range opts {
		opt(&options)
	}

	return options
}

// Option is for setting options.
type Option func(*Options)

// WithHandler sets handler.
func WithHandler(handler Handler) Option {
	return func(o *Options) {
		if handler != nil {
			o.Handler = handler
		}
	}
}

// WithLogger sets logger.
func WithLogger(logger Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithSlotNum sets slot number, must be greater than 0.
// If not, it will be ignored.
func WithSlotNum(num int) Option {
	return func(o *Options) {
		if num > 0 {
			o.SlotNum = num
		}
	}
}

// WithTickDuration sets tick duration, must be greater than 0.
// If not, it will be ignored.
func WithTickDuration(d time.Duration) Option {
	return func(o *Options) {
		if d > 0 {
			o.TickDuration = d
		}
	}
}
