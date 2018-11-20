package structures

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"log"
	"strings"

	"github.com/gelembjuk/oursql/lib/utils"
)

// TXOutput represents a transaction output
type TXCurrrencyOutput struct {
	Value      float64
	PubKeyHash []byte
}

// Simplified output format. To use externally
// It has all info in human readable format
// this can be used to display info abut outputs wihout references to transaction object
type TXOutputIndependent struct {
	Value          float64
	DestPubKeyHash []byte
	SendPubKeyHash []byte
	TXID           []byte
	OIndex         int
	IsBase         bool
	BlockHash      []byte
}

type TXOutputIndependentList []TXOutputIndependent

// Lock signs the output
func (out *TXCurrrencyOutput) Lock(address []byte) {
	pubKeyHash, err := utils.AddresBToPubKeyHash(address)

	if err != nil {
		// send to noone !
		// TODO some better behavior needed here
		out.PubKeyHash = []byte{}
	}
	out.PubKeyHash = pubKeyHash
}

// Lock signs the output
func (out TXCurrrencyOutput) HasOutAddress() bool {
	return len(out.PubKeyHash) > 0
}
// IsLockedWithKey checks if the output can be used by the owner of the pubkey
func (out *TXCurrrencyOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

// Same as IsLockedWithKey but for simpler structure
func (out *TXOutputIndependent) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.DestPubKeyHash, pubKeyHash) == 0
}

// build independed transaction from normal output
func (out *TXOutputIndependent) LoadFromSimple(sout TXCurrrencyOutput, txid []byte, ind int, sender []byte, iscoinbase bool, blockHash []byte) {
	out.OIndex = ind
	out.DestPubKeyHash = sout.PubKeyHash
	out.SendPubKeyHash = sender
	out.Value = sout.Value
	out.TXID = txid
	out.IsBase = iscoinbase
	out.BlockHash = blockHash
}

// NewTXOutput create a new TXOutput
func NewTXOutput(value float64, address string) *TXCurrrencyOutput {
	txo := &TXCurrrencyOutput{value, nil}
	txo.Lock([]byte(address))

	return txo
}

// TXOutputs collects TXOutput
type TXOutputs struct {
	Outputs []TXCurrrencyOutput
}

// Serialize serializes TXOutputs
func (outs TXOutputs) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// DeserializeOutputs deserializes TXOutputs
func DeserializeOutputs(data []byte) TXOutputs {
	var outputs TXOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	if err != nil {
		log.Panic(err)
	}

	return outputs
}

func (output TXCurrrencyOutput) String() string {
	lines := []string{}

	lines = append(lines, fmt.Sprintf("       Value:  %f", output.Value))
	lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))

	return strings.Join(lines, "\n")
}

func (output TXCurrrencyOutput) ToBytes() ([]byte, error) {
	buff := new(bytes.Buffer)

	err := binary.Write(buff, binary.BigEndian, output.Value)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buff, binary.BigEndian, output.PubKeyHash)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func (a TXOutputIndependentList) Len() int           { return len(a) }
func (a TXOutputIndependentList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TXOutputIndependentList) Less(i, j int) bool { return a[i].Value < a[j].Value }
