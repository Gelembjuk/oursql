package dbquery

import (
	"fmt"
)

const parseNoPrimaryKeyInCondError = "noprimarykeyincond"

type ParseError struct {
	errorSrt  string
	errorKind string
}

func (e *ParseError) Error() string {
	if e.errorKind == parseNoPrimaryKeyInCondError {
		return e.errorSrt
	}
	return fmt.Sprintf("Parse error: %s, kind: %s", e.errorSrt, e.errorKind)
}

func NewParseNoPrimaryKeyInCondError(str string) error {
	return &ParseError{str, parseNoPrimaryKeyInCondError}
}
