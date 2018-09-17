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

type CustomRequestActionInterface interface {
	getPacket() []byte
}

type customRequestModifyInterface interface {
	setOriginalRequest(p []byte)
}

type customRequestActionNeedProtocolInterface interface {
	setProtocolInfo(pi protocolInfo)
}

type CustomResponseKeyValue struct {
	Key   string
	Value string
}

type RequestQueryFilterCallback func(query string, sessionID string) (CustomRequestActionInterface, error)
type ResponseFilterCallback func(sessionID string, err error)

// Interface for a filter structure
// It is alternative for callbacks and can keep some state inside
type DBProxyFilter interface {
	RequestCallback(query string, sessionID string) (CustomRequestActionInterface, error)
	ResponseCallback(sessionID string, err error)
}

// Custom responses constructors
// Make new Custom Error Response
func NewCustomErrorResponse(err string, code uint16) CustomRequestActionInterface {
	r := customResponseError{}
	r.Code = code
	r.Message = err
	return &r
}
func NewCustomDataKeyValueResponse(rows []CustomResponseKeyValue) CustomRequestActionInterface {
	r := customResponseRowsKeyValues{}
	r.rows = rows
	return &r
}

func NewCustomOKResponse(ar uint) CustomRequestActionInterface {
	r := customResponseOK{}
	r.rowsUpdated = ar
	return &r
}

func NewCustomQueryRequest(query string) CustomRequestActionInterface {
	r := customResponseReplaceQuery{}
	r.replaceQuery = query
	return &r
}
