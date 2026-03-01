package cocart

import (
	"sync"
	"time"
)

// RequestEvent is emitted before an HTTP request is sent.
type RequestEvent struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

// ResponseEvent is emitted after an HTTP response is received.
type ResponseEvent struct {
	Method   string
	URL      string
	Status   int
	Duration time.Duration
}

// ErrorEvent is emitted when a request fails.
type ErrorEvent struct {
	Method string
	URL    string
	Err    error
}

// RetryEvent is emitted when a request is retried.
type RetryEvent struct {
	Method     string
	URL        string
	Attempt    int
	MaxRetries int
	Delay      time.Duration
	Reason     string
}

// AuthRefreshEvent is emitted when JWT token refresh is attempted.
type AuthRefreshEvent struct {
	Success bool
}

// eventEmitter manages typed event listeners.
type eventEmitter struct {
	mu              sync.RWMutex
	requestHandlers []func(RequestEvent)
	responseHandlers []func(ResponseEvent)
	errorHandlers   []func(ErrorEvent)
	retryHandlers   []func(RetryEvent)
	authRefreshHandlers []func(AuthRefreshEvent)
}

func newEventEmitter() *eventEmitter {
	return &eventEmitter{}
}

// OnRequest registers a handler for request events.
// Returns an unsubscribe function.
func (c *Client) OnRequest(fn func(RequestEvent)) func() {
	c.emitter.mu.Lock()
	defer c.emitter.mu.Unlock()
	c.emitter.requestHandlers = append(c.emitter.requestHandlers, fn)
	idx := len(c.emitter.requestHandlers) - 1
	return func() {
		c.emitter.mu.Lock()
		defer c.emitter.mu.Unlock()
		c.emitter.requestHandlers = removeHandler(c.emitter.requestHandlers, idx)
	}
}

// OnResponse registers a handler for response events.
// Returns an unsubscribe function.
func (c *Client) OnResponse(fn func(ResponseEvent)) func() {
	c.emitter.mu.Lock()
	defer c.emitter.mu.Unlock()
	c.emitter.responseHandlers = append(c.emitter.responseHandlers, fn)
	idx := len(c.emitter.responseHandlers) - 1
	return func() {
		c.emitter.mu.Lock()
		defer c.emitter.mu.Unlock()
		c.emitter.responseHandlers = removeHandler(c.emitter.responseHandlers, idx)
	}
}

// OnError registers a handler for error events.
// Returns an unsubscribe function.
func (c *Client) OnError(fn func(ErrorEvent)) func() {
	c.emitter.mu.Lock()
	defer c.emitter.mu.Unlock()
	c.emitter.errorHandlers = append(c.emitter.errorHandlers, fn)
	idx := len(c.emitter.errorHandlers) - 1
	return func() {
		c.emitter.mu.Lock()
		defer c.emitter.mu.Unlock()
		c.emitter.errorHandlers = removeHandler(c.emitter.errorHandlers, idx)
	}
}

// OnRetry registers a handler for retry events.
// Returns an unsubscribe function.
func (c *Client) OnRetry(fn func(RetryEvent)) func() {
	c.emitter.mu.Lock()
	defer c.emitter.mu.Unlock()
	c.emitter.retryHandlers = append(c.emitter.retryHandlers, fn)
	idx := len(c.emitter.retryHandlers) - 1
	return func() {
		c.emitter.mu.Lock()
		defer c.emitter.mu.Unlock()
		c.emitter.retryHandlers = removeHandler(c.emitter.retryHandlers, idx)
	}
}

// OnAuthRefresh registers a handler for auth refresh events.
// Returns an unsubscribe function.
func (c *Client) OnAuthRefresh(fn func(AuthRefreshEvent)) func() {
	c.emitter.mu.Lock()
	defer c.emitter.mu.Unlock()
	c.emitter.authRefreshHandlers = append(c.emitter.authRefreshHandlers, fn)
	idx := len(c.emitter.authRefreshHandlers) - 1
	return func() {
		c.emitter.mu.Lock()
		defer c.emitter.mu.Unlock()
		c.emitter.authRefreshHandlers = removeHandler(c.emitter.authRefreshHandlers, idx)
	}
}

func (e *eventEmitter) emitRequest(evt RequestEvent) {
	e.mu.RLock()
	handlers := make([]func(RequestEvent), len(e.requestHandlers))
	copy(handlers, e.requestHandlers)
	e.mu.RUnlock()
	for _, fn := range handlers {
		func() {
			defer func() { recover() }()
			fn(evt)
		}()
	}
}

func (e *eventEmitter) emitResponse(evt ResponseEvent) {
	e.mu.RLock()
	handlers := make([]func(ResponseEvent), len(e.responseHandlers))
	copy(handlers, e.responseHandlers)
	e.mu.RUnlock()
	for _, fn := range handlers {
		func() {
			defer func() { recover() }()
			fn(evt)
		}()
	}
}

func (e *eventEmitter) emitError(evt ErrorEvent) {
	e.mu.RLock()
	handlers := make([]func(ErrorEvent), len(e.errorHandlers))
	copy(handlers, e.errorHandlers)
	e.mu.RUnlock()
	for _, fn := range handlers {
		func() {
			defer func() { recover() }()
			fn(evt)
		}()
	}
}

func (e *eventEmitter) emitRetry(evt RetryEvent) {
	e.mu.RLock()
	handlers := make([]func(RetryEvent), len(e.retryHandlers))
	copy(handlers, e.retryHandlers)
	e.mu.RUnlock()
	for _, fn := range handlers {
		func() {
			defer func() { recover() }()
			fn(evt)
		}()
	}
}

func (e *eventEmitter) emitAuthRefresh(evt AuthRefreshEvent) {
	e.mu.RLock()
	handlers := make([]func(AuthRefreshEvent), len(e.authRefreshHandlers))
	copy(handlers, e.authRefreshHandlers)
	e.mu.RUnlock()
	for _, fn := range handlers {
		func() {
			defer func() { recover() }()
			fn(evt)
		}()
	}
}

func removeHandler[T any](slice []T, idx int) []T {
	if idx < 0 || idx >= len(slice) {
		return slice
	}
	return append(slice[:idx], slice[idx+1:]...)
}
