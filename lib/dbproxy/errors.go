package dbproxy

import (
	"fmt"
)

const dbProxyConfigError = "rownotfound"

type DBProxyError struct {
	err  string
	kind string
}

func (e DBProxyError) Error() string {
	if e.kind == dbProxyConfigError {
		return fmt.Sprintf("DB Proxy Config Error: %s", e.err)
	}
	return fmt.Sprintf("DB Proxy Error: %s", e.err)
}

func (e DBProxyError) Kind() string {
	return e.kind
}

func (e DBProxyError) IsDBProxyConfigError() bool {
	return e.kind == dbProxyConfigError
}

func newDBError(err string, kind string) error {
	return &DBProxyError{err, kind}
}

func NewConfigDBProxyError(err string) error {
	return &DBProxyError{err, dbProxyConfigError}
}
