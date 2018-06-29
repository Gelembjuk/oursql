package database

// Custom errors

import (
	"fmt"
)

const TXVerifyErrorNoInput = "noinput"
const DBCursorBreak = "cursorbreak"
const DBHashNotFoundError = "hashnotfound"
const DBHashEmptyError = "hashisemptyd"
const DBHashError = "hashemptyd"

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
