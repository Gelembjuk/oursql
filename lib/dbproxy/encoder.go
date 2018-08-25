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

	length := make([]byte, 4)
	binary.LittleEndian.PutUint32(length, uint32(payloadLen)) // problem can be if length is more uint16

	// TODO if length is too big, try to truncate error message

	res := []byte{length[0], length[1], length[2], 1, responseErr, bs[0], bs[1]}

	res = append(res, errBytes...)
	return res
}

func (e ResponseError) Error() string {

	return e.Message
}
