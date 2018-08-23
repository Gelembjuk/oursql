package dbproxy

import (
	"log"
)

type DBProxyInterface interface {
	SetCallbacks(requestCallback RequestQueryFilterCallback, responseCallback ResponseFilterCallback)
	SetFilter(filterObj DBProxyFilter)
	SetLoggers(t *log.Logger, e *log.Logger)
	Init() error
	Run() error // this function should start new goroutine
	IsStopped() bool
	WaitStop()
	Stop() error
}

type RequestQueryFilterCallback func(query string, sessionID string) error
type ResponseFilterCallback func(sessionID string, err error)

// Interface for a filter structure
// It is alternative for callbacks and can keep some state inside
type DBProxyFilter interface {
	RequestCallback(query string, sessionID string) error
	ResponseCallback(sessionID string, err error)
}
