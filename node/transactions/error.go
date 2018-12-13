package transactions

// Custom errors

import (
	"fmt"
)

const TXVerifyErrorNoInput = "noinput"
const TXNotFoundErrorUnspent = "inunspent"
const TXSQLBaseDifferentError = "sqlbaseisdifferent"
const TXPrepareNoFundsError = "noenoughfunds"
const txPoolCacheNoMemoryError = "noenoughfunds"

type TXVerifyError struct {
	err  string
	kind string
	TX   []byte
}

type TXNotFoundError struct {
	err  string
	kind string
}

type TXPrepareError struct {
	err  string
	kind string
}

type txPoolError struct {
	err  string
	kind string
}

func (e TXVerifyError) Error() string {
	return fmt.Sprintf("Transaction verify failed: %s, for TX %x", e.err, e.TX)
}

func (e *TXNotFoundError) Error() string {
	return fmt.Sprintf("Transaction verify failed: %s, kind: %s", e.err, e.kind)
}

func (e *TXPrepareError) Error() string {
	return fmt.Sprintf("Transaction prepare failed: %s, kind: %s", e.err, e.kind)
}

func (e *txPoolError) Error() string {
	return fmt.Sprintf("Pool Error: %s, kind: %s", e.err, e.kind)
}

func (e *TXPrepareError) ErrorOrig() string {
	return e.err
}

func (e TXVerifyError) GetKind() string {
	return e.kind
}

func (e TXVerifyError) IsKind(kind string) bool {
	return e.kind == kind
}

func (e *TXNotFoundError) GetKind() string {
	return e.kind
}
func (e *TXPrepareError) GetKind() string {
	return e.kind
}
func (e *txPoolError) isNoMemory() bool {
	return e.kind == txPoolCacheNoMemoryError
}

func NewTXVerifyError(err string, kind string, TX []byte) error {
	return &TXVerifyError{err, kind, TX}
}

func NewTXVerifySQLBaseError(err string, TX []byte) error {
	return &TXVerifyError{err, TXSQLBaseDifferentError, TX}
}

func NewTXNotFoundError(err string, kind string) error {
	return &TXNotFoundError{err, kind}
}
func NewTXNotFoundUOTError(err string) error {
	return &TXNotFoundError{err, TXNotFoundErrorUnspent}
}
func NewTXNoEnoughFundsdError(err string) error {
	return &TXPrepareError{err, TXPrepareNoFundsError}
}

// Generate when pool memory cache can not be used because no memory
func newTXPoolCacheNoMemoryError(err string) error {
	return &txPoolError{err, txPoolCacheNoMemoryError}
}
