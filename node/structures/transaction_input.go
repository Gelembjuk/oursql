package structures

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)
import "github.com/gelembjuk/oursql/lib/utils"

// TXInput represents a transaction input
type TXCurrencyInput struct {
	Txid      []byte
	Vout      int
	Signature []byte
	PubKey    []byte // this is the wallet who spends transaction
}

// UsesKey checks whether the address initiated the transaction
func (in *TXCurrencyInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash, _ := utils.HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

func (input TXCurrencyInput) String() string {
	lines := []string{}

	lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
	lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
	lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
	lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))

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

	err = binary.Write(buff, binary.BigEndian, input.Signature)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buff, binary.BigEndian, input.PubKey)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}
