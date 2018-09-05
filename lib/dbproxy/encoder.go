package dbproxy

import (
	"bytes"
	"encoding/binary"
)

// Encode custom responses by a proxy

type ResponseError struct {
	Message string
	Code    uint16
}

type responseRowsKeyValues struct {
	rows    []CustomResponseKeyValue
	counter uint
}

func NewMySQLError(err string, code uint16) ResponseError {
	return ResponseError{err, code}
}

func newMySQLDataKeyValues(rows []CustomResponseKeyValue) responseRowsKeyValues {
	r := responseRowsKeyValues{}
	r.rows = rows
	return r
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

// Prepare response packet. which contains list of rows
// https://github.com/siddontang/mixer/blob/master/doc/mysql-proxy/protocol.txt
// https://dev.mysql.com/doc/internals/en/com-query-response.html
func (r *responseRowsKeyValues) getPacket() []byte {
	r.counter = 0

	var b bytes.Buffer

	b.Write(r.completePacket([]byte{2})) // 2 columns

	b.Write(r.getColumnDefPacket("BC", "CustomResponse", "Key"))
	b.Write(r.getColumnDefPacket("BC", "CustomResponse", "Value"))
	//b.Write(r.getColumnDefPacket("dbtest", "list", "Key"))
	//b.Write(r.getColumnDefPacket("dbtest", "list", "Value"))

	// send rows
	for _, row := range r.rows {
		b.Write(r.getRowData(row.Key, row.Value))
	}

	b.Write(r.completePacket([]byte{0xfe, 0x00, 0x00, 0x22, 0x00, 0x00, 0x00})) // EOF

	return b.Bytes()
}

// make a data to be a packet in a sequence
func (r *responseRowsKeyValues) completePacket(data []byte) []byte {
	r.counter = r.counter + 1

	length := make([]byte, 4)
	binary.LittleEndian.PutUint32(length, uint32(len(data)))

	res := []byte{length[0], length[1], length[2], uint8(r.counter)}

	res = append(res, data...)

	return res
}

func (r *responseRowsKeyValues) getColumnDefPacket(schema, table, column string) []byte {
	var b bytes.Buffer

	b.Write(r.getLengEncStr("def"))

	b.Write(r.getLengEncStr(schema))

	b.Write(r.getLengEncStr(table))

	b.Write(r.getLengEncStr(table))

	b.Write(r.getLengEncStr(column))

	b.Write(r.getLengEncStr(column))

	b.WriteByte(0x0c)

	b.Write([]byte{0x21, 0x00, 0xfd, 0xff, 0x02, 0x00}) // charset and max length

	b.WriteByte(0xfc) // type MYSQL_TYPE_BLOB

	b.Write([]byte{0x10, 0x00, 0x00, 0x00, 0x00})

	return r.completePacket(b.Bytes())
}

// Returns length encoded string for MySQL protocol
func (r *responseRowsKeyValues) getLengEncStr(data string) []byte {
	str := []byte(data)

	length := len(str)

	res := []byte{}

	// we don't expect to have strings longer 65000
	if length > 251 {
		lb := make([]byte, 4)
		binary.LittleEndian.PutUint32(lb, uint32(length))
		res = []byte{0xfc, lb[0], lb[1]}
	} else {
		res = append(res, uint8(length))
	}

	res = append(res, str...)

	return res
}

// Create row packet
func (r *responseRowsKeyValues) getRowData(val1, val2 string) []byte {
	row := r.getLengEncStr(val1)

	row = append(row, r.getLengEncStr(val2)...)

	return r.completePacket(row)
}
