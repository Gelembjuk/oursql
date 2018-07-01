package structures

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"time"

	"encoding/gob"
	"fmt"

	"github.com/gelembjuk/oursql/lib"
	"github.com/gelembjuk/oursql/lib/utils"
)

// CurrencyTransaction represents a Bitcoin transaction
type CurrencyTransaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
	Time int64
}

// Return ID of transaction
func (tx CurrencyTransaction) GetID() []byte {
	return tx.ID
}

// IsCoinbase checks whether the transaction is currency transaction
func (tx CurrencyTransaction) CheckTypeIs(t string) bool {
	if t != TXTypeCurrency {
		return false
	}
	return true
}

// IsCoinbase checks whether the transaction is coinbase
func (tx CurrencyTransaction) CheckSubTypeIs(t string) bool {
	cb := tx.isCoinbase()

	if t == TXTypeCurrencyCoinbase && cb {
		return true
	}
	if t == TXTypeCurrencyTransfer && !cb {
		return true
	}

	return false
}

// check if TX is coin base
func (tx CurrencyTransaction) isCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// check if needs signatures. in case of coin base doesn't need it
func (tx CurrencyTransaction) NeedsSignatures() bool {
	return !tx.isCoinbase()
}

// Hash returns the hash of the CurrencyTransaction
func (tx *CurrencyTransaction) timeNow() {
	tx.Time = time.Now().UTC().UnixNano()
}

// Hash returns the hash of the CurrencyTransaction
func (tx *CurrencyTransaction) Hash() ([]byte, error) {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	txser, err := txCopy.serialize()

	if err != nil {
		return nil, err
	}

	hash = sha256.Sum256(txser)

	tx.ID = hash[:]
	return tx.ID, nil
}

// String returns a human-readable representation of a transaction
func (tx CurrencyTransaction) String() string {
	var lines []string
	from := "Coin Base"
	fromhash := []byte{}

	if !tx.isCoinbase() {
		from, _ = utils.PubKeyToAddres(tx.Vin[0].PubKey)
		fromhash, _ = utils.HashPubKey(tx.Vin[0].PubKey)
	}

	to := ""
	amount := 0.0

	for _, output := range tx.Vout {
		if bytes.Compare(fromhash, output.PubKeyHash) != 0 {
			to, _ = utils.PubKeyHashToAddres(output.PubKeyHash)
			amount = output.Value
			break
		}
	}

	lines = append(lines, fmt.Sprintf("--- CurrencyTransaction %x:", tx.ID))
	lines = append(lines, fmt.Sprintf("    FROM %s TO %s VALUE %f", from, to, amount))
	lines = append(lines, fmt.Sprintf("    Time %d (%s)", tx.Time, time.Unix(0, tx.Time)))

	if !tx.isCoinbase() {
		for i, input := range tx.Vin {
			address, _ := utils.PubKeyToAddres(input.PubKey)
			lines = append(lines, fmt.Sprintf("     Input %d:", i))
			lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
			lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
			lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
			lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
			lines = append(lines, fmt.Sprintf("       Address:   %s", address))
		}
	}

	for i, output := range tx.Vout {
		address, _ := utils.PubKeyHashToAddres(output.PubKeyHash)
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %f", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
		lines = append(lines, fmt.Sprintf("       Address: %s", address))
	}

	return strings.Join(lines, "\n")
}

// TrimmedCopy creates a trimmed copy of CurrencyTransaction to be used in signing
func (tx *CurrencyTransaction) trimmedCopy() CurrencyTransaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, nil, nil})
	}

	for _, vout := range tx.Vout {
		pkh := utils.CopyBytes(vout.PubKeyHash)

		outputs = append(outputs, TXOutput{vout.Value, pkh})
	}
	txID := utils.CopyBytes(tx.ID)
	txCopy := CurrencyTransaction{txID, inputs, outputs, tx.Time}

	return txCopy
}

// TrimmedCopy creates a trimmed copy of CurrencyTransaction to be used in signing
func (tx CurrencyTransaction) Copy() (TransactionInterface, error) {
	//if tx.IsCoinbase() {
	//	return *tx, nil
	//}
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		sig := utils.CopyBytes(vin.Signature)

		pk := utils.CopyBytes(vin.PubKey)

		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, sig, pk})
	}

	for _, vout := range tx.Vout {
		pkh := utils.CopyBytes(vout.PubKeyHash)

		outputs = append(outputs, TXOutput{vout.Value, pkh})
	}

	txID := utils.CopyBytes(tx.ID)

	txCopy := &CurrencyTransaction{txID, inputs, outputs, tx.Time}

	return txCopy, nil
}

// prepare data to sign as part of transaction
// this return slice of slices. Every of them must be signed for each TX Input
func (tx *CurrencyTransaction) PrepareSignData(prevTXs map[int]TransactionInterface) ([][]byte, error) {
	tx.timeNow()

	for vinInd, vin := range tx.Vin {
		if _, ok := prevTXs[vinInd]; !ok {
			return nil, errors.New("Previous transaction is not correct")
		}
		if bytes.Compare(prevTXs[vinInd].GetID(), vin.Txid) != 0 {
			return nil, errors.New("Previous transaction is not correct")
		}
	}

	signdata := make([][]byte, len(tx.Vin))

	txCopy := tx.trimmedCopy()
	txCopy.ID = []byte{}

	for inID, _ := range txCopy.Vin {
		txCopy.Vin[inID].Signature = nil
	}

	for inID, vin := range txCopy.Vin {
		prevTx, ok := prevTXs[inID].(*CurrencyTransaction)

		if !ok {
			return nil, errors.New("Wring type of previous transaction")
		}

		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash

		signdata[inID], _ = txCopy.ToBytes()

		txCopy.Vin[inID].PubKey = nil
	}

	return signdata, nil
}

// Sets signatures for inputs. Signatures were created separately for data set prepared before
// in the function PrepareSignData
func (tx *CurrencyTransaction) SetSignatures(signatures [][]byte) error {

	for inID, _ := range tx.Vin {

		tx.Vin[inID].Signature = signatures[inID]
	}
	// when transaction is complete, we can add ID to it
	tx.Hash()

	return nil
}

// Verify verifies signatures of CurrencyTransaction inputs
// And total amount of inputs and outputs
func (tx *CurrencyTransaction) Verify(prevTXs map[int]TransactionInterface) error {
	if tx.isCoinbase() {
		// coinbase has only 1 output and it must have value equal to constant
		if tx.Vout[0].Value != lib.CurrencyPaymentForBlockMade {
			return errors.New("Value of coinbase transaction is wrong")
		}
		if len(tx.Vout) > 1 {
			return errors.New("Coinbase transaction can have only 1 output")
		}
		return nil
	}
	// calculate total input
	totalinput := float64(0)

	for vind, vin := range tx.Vin {
		prevTx, ok := prevTXs[vind].(*CurrencyTransaction)

		if !ok {
			return errors.New("Wring type of previous transaction")
		}

		if prevTx.ID == nil {
			return errors.New("Previous transaction is not correct")
		}
		amount := prevTx.Vout[vin.Vout].Value
		totalinput += amount
	}

	txCopy := tx.trimmedCopy()
	txCopy.ID = []byte{}

	for inID, _ := range tx.Vin {
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = nil
	}

	for inID, vin := range tx.Vin {
		// full input transaction

		prevTx, ok := prevTXs[inID].(*CurrencyTransaction)

		if !ok {
			return errors.New("Wring type of previous transaction")
		}

		//hash of key who signed this input
		signPubKeyHash, _ := utils.HashPubKey(vin.PubKey)

		if bytes.Compare(prevTx.Vout[vin.Vout].PubKeyHash, signPubKeyHash) != 0 {
			return errors.New(fmt.Sprintf("Sign Key Hash for input %x is different from output hash", vin.Txid))
		}

		// replace pub key with its hash. same was done when signing
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash

		dataToVerify, _ := txCopy.ToBytes()

		v, err := utils.VerifySignature(vin.Signature, dataToVerify, vin.PubKey)

		if err != nil {
			return err
		}

		if !v {
			return errors.New(fmt.Sprintf("Signatire doe not match for input TX %x.", vin.Txid))
		}
		txCopy.Vin[inID].PubKey = nil
	}

	// calculate total output of transaction
	totaloutput := float64(0)

	for _, vout := range tx.Vout {
		if vout.Value < lib.CurrencySmallestUnit {
			return errors.New(fmt.Sprintf("Too small output value %f", vout.Value))
		}
		totaloutput += vout.Value
	}

	if math.Abs(totalinput-totaloutput) >= lib.CurrencySmallestUnit {
		return errors.New(fmt.Sprintf("Input and output values of a transaction are not same: %.10f vs %.10f . Diff %.10f", totalinput, totaloutput, totalinput-totaloutput))
	}

	return nil
}

// Serialize returns a serialized CurrencyTransaction
func (tx CurrencyTransaction) serialize() ([]byte, error) {
	// to remove any references to other ponters
	// do full copy of the TX

	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		return nil, err
	}

	return encoded.Bytes(), nil
}

// DeserializeCurrencyTransaction deserializes a transaction
func (tx *CurrencyTransaction) DeserializeTransaction(data []byte) error {
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(tx)

	if err != nil {
		return err
	}
	return nil
}

// converts transaction to slice of bytes
// this will be used to do a hash of transactions
func (tx CurrencyTransaction) ToBytes() ([]byte, error) {
	buff := new(bytes.Buffer)

	err := binary.Write(buff, binary.BigEndian, tx.ID)

	if err != nil {
		return nil, err
	}

	for _, vin := range tx.Vin {
		b, err := vin.ToBytes()
		if err != nil {
			return nil, err
		}

		err = binary.Write(buff, binary.BigEndian, b)
		if err != nil {
			return nil, err
		}
	}

	for _, vout := range tx.Vout {
		b, err := vout.ToBytes()
		if err != nil {
			return nil, err
		}
		err = binary.Write(buff, binary.BigEndian, b)
		if err != nil {
			return nil, err
		}
	}

	err = binary.Write(buff, binary.BigEndian, tx.Time)

	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func (tx CurrencyTransaction) IsComplete() bool {
	if tx.ID != nil {
		return true
	}
	return false
}

func (tx CurrencyTransaction) GetTime() int64 {
	return tx.Time
}
