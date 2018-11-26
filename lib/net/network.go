package net

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const Protocol = "tcp"
const NodeVersion = 1
const CommandLength = 12
const AuthStringLength = 20

// Represents a node address
type NodeAddr struct {
	Host                     string
	Port                     int
	SuccessConnections       uint
	FailedConnections        uint
	SuccessIncomeConnections uint
}

type NodeAddrShort struct {
	Host []byte
	Port int
}

func NewNodeAddr(host string, port int) NodeAddr {
	a := NodeAddr{}
	a.Host = host
	a.Port = port
	return a
}

// Convert to string in format host:port
func (n NodeAddr) NodeAddrToString() string {
	return n.String()
}

// Convert to string in format host:port
func (n NodeAddr) String() string {
	return n.Host + ":" + strconv.Itoa(n.Port)
}

// Notify this address got success attempt to connect
func (n *NodeAddr) ReportSuccessConn() {
	n.SuccessConnections = n.SuccessConnections + 1
}

// Get short format of the node
func (n NodeAddr) GetShortFormat() NodeAddrShort {
	ns := NodeAddrShort{}
	ns.Host = []byte(n.Host)
	ns.Port = n.Port
	return ns
}

// Notify this address got failed attempt to connect
func (n *NodeAddr) ReportFailedConn() {
	n.FailedConnections = n.FailedConnections + 1
}

// init a record after restore from DB
func (n *NodeAddr) InitAfterRestore() {
	n.FailedConnections = 0
	n.SuccessConnections = 0
}

// Compare to other node address if is same
func (n NodeAddr) CompareToAddress(addr NodeAddr) bool {
	h1 := strings.Trim(addr.Host, " ")
	h2 := strings.Trim(n.Host, " ")

	if h1 == "localhost" {
		h1 = "127.0.0.1"
	}
	if h2 == "localhost" {
		h2 = "127.0.0.1"
	}

	return (h1 == h2 && addr.Port == n.Port)
}

// Parse from string
func (n *NodeAddr) LoadFromString(addr string) error {
	parts := strings.SplitN(addr, ":", 2)

	if len(parts) < 2 {
		return errors.New("Wrong address")
	}
	n.Host = parts[0]

	port, err := strconv.Atoi(parts[1])

	if err != nil {
		return err
	}
	n.Port = port
	return nil
}

// Converts a command to bytes in fixed length
func CommandToBytes(command string) []byte {
	var bytes [CommandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

// Convert bytes back to command
func BytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

// Get command part from request string
func ExtractCommand(request []byte) []byte {
	return request[:CommandLength]
}

// Encode structure to bytes
func GobEncode(data interface{}) ([]byte, error) {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		return []byte{}, err
	}

	return buff.Bytes(), nil
}
