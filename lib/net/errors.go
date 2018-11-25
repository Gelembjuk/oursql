package net

import (
	"fmt"
)

const (
	errorCanNotConnect       = "cannotconnect"
	errorCanNotSend          = "cannotsend"
	errorNoResponse          = "noresponse"
	errorCanNotParseResponse = "cannotparseresponse"
)

type NetworkError struct {
	errStr string
	kind   string
}

func (e NetworkError) Error() string {
	if e.kind == errorCanNotConnect {
		return fmt.Sprintf("Network Connection Error: %s", e.errStr)
	}
	if e.kind == errorCanNotSend {
		return fmt.Sprintf("Network Data Transfer Error: %s", e.errStr)
	}
	if e.kind == errorNoResponse {
		return fmt.Sprintf("No Network Response: %s", e.errStr)
	}
	if e.kind == errorCanNotParseResponse {
		return fmt.Sprintf("Can Not Parse Network Response: %s", e.errStr)
	}
	return fmt.Sprintf("Network Error: %s", e.errStr)
}

func (e NetworkError) WasConnFailure() bool {
	if e.kind == errorCanNotConnect {
		return true
	}
	return false
}

func NewCanNotConnectError(err string) error {
	return &NetworkError{err, errorCanNotConnect}
}

func NewCanNotSendError(err string) error {
	return &NetworkError{err, errorCanNotSend}
}

func NewNoResponseError(err string) error {
	return &NetworkError{err, errorNoResponse}
}

func NewCanNotParseResponseError(err string) error {
	return &NetworkError{err, errorCanNotParseResponse}
}
