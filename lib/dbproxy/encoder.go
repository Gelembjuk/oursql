package dbproxy

import (
	"encoding/binary"
)

// Encode custom responses by a proxy

type ResponseError struct {
	Message string
	Code    uint16
}

func NewMySQLError(err string, code uint16) ResponseError {
	return ResponseError{err, code}
}

func (e ResponseError) getMySQLError() []byte {
	errBytes := []byte(e.Message)
	payloadLen := len(errBytes) + 3

	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, e.Code)

	res := []byte{byte(payloadLen), 0, 0, 1, responseErr, bs[0], bs[1]}

	res = append(res, errBytes...)
	return res
}

func (e ResponseError) Error() string {

	return e.Message
}
