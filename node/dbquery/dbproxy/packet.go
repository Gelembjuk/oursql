package dbproxy

/**
* This code is part of the package github.com/orderbynull/lottip/tree/master/protocol
 */

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
)

//...
type connSettings struct {
	ClientCapabilities uint32
	ServerCapabilities uint32
	SelectedDb         string
}

//...
func (h *connSettings) deprecateEOFSet() bool {
	return ((clientDeprecateEOF & h.ServerCapabilities) != 0) &&
		((clientDeprecateEOF & h.ClientCapabilities) != 0)
}

// ProcessHandshake handles handshake between server and client.
// Returns server and client handshake responses
func processHandshake(client net.Conn, mysql net.Conn) (*handshakeV10, *handshakeResponse41, error) {

	// Read server handshake
	packet, err := proxyPacket(mysql, client)
	if err != nil {
		println(err.Error())
		return nil, nil, err
	}

	serverHandshake, err := decodeHandshakeV10(packet)
	if err != nil {
		println(err.Error())
		return nil, nil, err
	}

	// Read client handshake response
	packet, err = proxyPacket(client, mysql)
	if err != nil {
		println(err.Error())
		return nil, nil, err
	}

	clientHandshake, err := decodeHandshakeResponse41(packet)
	if err != nil {
		println(err.Error())
		return nil, nil, err
	}

	// Read server OK response
	if _, err = proxyPacket(mysql, client); err != nil {
		println(err.Error())
		return nil, nil, err
	}

	return serverHandshake, clientHandshake, nil
}

// ReadPrepareResponse reads response from MySQL server for COM_STMT_PREPARE
// query issued by client.
// ...
func readPrepareResponse(conn net.Conn) ([]byte, byte, error) {
	pkt, err := readPacket(conn)
	if err != nil {
		return nil, 0, err
	}

	switch pkt[4] {
	case responsePrepareOk:
		numParams := binary.LittleEndian.Uint16(pkt[9:11])
		numColumns := binary.LittleEndian.Uint16(pkt[11:13])
		packetsExpected := 0

		if numParams > 0 {
			packetsExpected += int(numParams) + 1
		}

		if numColumns > 0 {
			packetsExpected += int(numColumns) + 1
		}

		var data []byte
		var eofCnt int

		data = append(data, pkt...)

		for i := 1; i <= packetsExpected; i++ {
			eofCnt++
			pkt, err = readPacket(conn)
			if err != nil {
				return nil, 0, err
			}

			data = append(data, pkt...)
		}

		return data, responseOk, nil

	case responseErr:
		return pkt, responseErr, nil
	}

	return nil, 0, nil
}

func readErrMessage(errPacket []byte) string {
	return string(errPacket[13:])
}

func readShowFieldsResponse(conn net.Conn) ([]byte, byte, error) {
	return readResponse(conn, true)
}

// ReadResponse ...
func readResponse(conn net.Conn, deprecateEof bool) ([]byte, byte, error) {
	pkt, err := readPacket(conn)
	if err != nil {
		return nil, 0, err
	}

	switch pkt[4] {
	case responseOk:
		return pkt, responseOk, nil

	case responseErr:
		return pkt, responseErr, nil

	case responseLocalinfile:
	}

	var data []byte

	data = append(data, pkt...)

	if !deprecateEof {
		pktReader := bytes.NewReader(pkt[4:])
		columns, _ := readLenEncodedInteger(pktReader)

		toRead := int(columns) + 1

		for i := 0; i < toRead; i++ {
			pkt, err := readPacket(conn)
			if err != nil {
				return nil, 0, err
			}

			data = append(data, pkt...)
		}
	}

	for {
		pkt, err := readPacket(conn)
		if err != nil {
			return nil, 0, err
		}

		data = append(data, pkt...)

		if pkt[4] == responseEof {
			break
		}
	}

	return data, responseResultset, nil
}

// ReadPacket ...
func readPacket(conn net.Conn) ([]byte, error) {

	// Read packet header
	header := []byte{0, 0, 0, 0}
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	// Calculate packet body length
	bodyLen := int(uint32(header[0]) | uint32(header[1])<<8 | uint32(header[2])<<16)

	// Read packet body
	body := make([]byte, bodyLen)
	n, err := io.ReadFull(conn, body)
	if err != nil {
		return nil, err
	}

	return append(header, body[0:n]...), nil
}

// WritePacket ...
func writePacket(pkt []byte, conn net.Conn) (int, error) {
	n, err := conn.Write(pkt)
	if err != nil {
		return 0, err
	}

	return n, nil
}

// ProxyPacket ...
func proxyPacket(src, dst net.Conn) ([]byte, error) {
	pkt, err := readPacket(src)
	if err != nil {
		return nil, err
	}

	_, err = writePacket(pkt, dst)
	if err != nil {
		return nil, err
	}

	return pkt, nil
}
