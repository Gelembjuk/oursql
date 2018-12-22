package database

// Custom errors

import (
	"fmt"
)

const (
	TXVerifyErrorNoInput = "noinput"
	DBCursorBreak        = "cursorbreak"
	DBHashNotFoundError  = "hashnotfound"
	DBHashEmptyError     = "hashisemptyd"
	DBHashError          = "hashemptyd"
	DBRowNotFoundError   = "rownotfound"
	DBConfigError        = "config"
	DBTableNotFound      = "tablenotfound"
)

type DBError struct {
	err  string
	kind string
}

func (e *DBError) Error() string {
	return fmt.Sprintf("Database Error: %s", e.err)
}

func (e *DBError) Kind() string {
	return e.kind
}

func (e *DBError) IsKind(kind string) bool {
	return e.kind == kind
}

// Check if this is row not found error
func (e *DBError) IsRowNotFound() bool {
	return e.kind == DBRowNotFoundError
}

// Check if this Table Not Found error
func (e *DBError) IsTableNotFound() bool {
	return e.kind == DBTableNotFound
}

func NewDBError(err string, kind string) error {
	return &DBError{err, kind}
}

func NewBucketNotFoundDBError() error {
	return &DBError{"Bucket is not found", "bucket"}
}

func NewNotFoundDBError(kind string) error {
	return &DBError{"Not found", kind}
}

func NewDBIsNotReadyError() error {
	return &DBError{"Database is not ready", "database"}
}

func NewDBCursorStopError() error {
	return &DBError{"Break data loop", DBCursorBreak}
}

func NewHashNotFoundDBError(err string) error {
	if err == "" {
		err = "Block hash is not found"
	}
	return &DBError{err, DBHashNotFoundError}
}

func NewHashEmptyDBError() error {
	return &DBError{"Provided hash is empty", DBHashEmptyError}
}

func NewHashDBError(err string) error {
	return &DBError{err, DBHashError}
}

func NewRowNotFoundDBError(err string) error {
	return &DBError{err, DBRowNotFoundError}
}

func NewTableNotFoundDBError(err string) error {
	return &DBError{err, DBTableNotFound}
}

func NewConfigDBError(err string) error {
	return &DBError{err, DBConfigError}
}
