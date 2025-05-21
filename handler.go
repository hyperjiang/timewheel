package timewheel

import "log"

// Handler is the handler interface for the time wheel.
type Handler interface {
	Handle(param any)
}

// HandlerFunc is a function type that implements the Handler interface.
type HandlerFunc func(param any)

func (f HandlerFunc) Handle(param any) {
	f(param)
}

// NewHandlerFunc creates a new HandlerFunc.
// It is a convenience function to create a Handler from a function.
// It is useful when you want to use a function as a handler without defining a separate struct.
func NewHandlerFunc(f HandlerFunc) Handler {
	return HandlerFunc(f)
}

var defaultHandler = NewHandlerFunc(func(param any) {
	log.Printf("handling %v\n", param)
})
