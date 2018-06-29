package transactions

// Custom errors

import (
	"fmt"
)

const TXVerifyErrorNoInput = "noinput"
const TXNotFoundErrorUnspent = "inunspent"

type TXVerifyError struct {
	err  string
	kind string
	TX   []byte
}

type TXNotFoundError struct {
	err  string
	kind string
}

func (e *TXVerifyError) Error() string {
	return fmt.Sprintf("Transaction verify failed: %s, for TX %x", e.err, e.TX)
}

func (e *TXNotFoundError) Error() string {
	return fmt.Sprintf("Transaction verify failed: %s, kinf: %s", e.err, e.kind)
}

func (e *TXVerifyError) GetKind() string {
	return e.kind
}

func (e *TXNotFoundError) GetKind() string {
	return e.kind
}

func NewTXVerifyError(err string, kind string, TX []byte) error {
	return &TXVerifyError{err, kind, TX}
}

func NewTXNotFoundError(err string, kind string) error {
	return &TXNotFoundError{err, kind}
}
func NewTXNotFoundUOTError(err string) error {
	return &TXNotFoundError{err, TXNotFoundErrorUnspent}
}
