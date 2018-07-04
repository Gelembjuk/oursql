package structures

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

// TXInput represents a transaction input
type TXCurrencyInput struct {
	Txid []byte
	Vout int
}

func (input TXCurrencyInput) String() string {
	lines := []string{}

	lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
	lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))

	return strings.Join(lines, "\n")
}

func (input TXCurrencyInput) ToBytes() ([]byte, error) {
	buff := new(bytes.Buffer)

	err := binary.Write(buff, binary.BigEndian, input.Txid)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buff, binary.BigEndian, int32(input.Vout))
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}
