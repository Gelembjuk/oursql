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

// Transaction represents a Bitcoin transaction
type Transaction struct {
	ID         []byte
	Time       int64
	Signature  []byte // one signature for full transaction
	ByPubKey   []byte
	Vin        []TXCurrencyInput
	Vout       []TXCurrrencyOutput
	SQLCommand SQLUpdate
	SQLBaseTX  []byte // ID of transaction where same row was affected last time
}

// execute when new tranaction object is created
func (tx *Transaction) initNewTX() {
	tx.Time = time.Now().UTC().UnixNano()
}

// execute when new TX is ready. all is set. do last action
func (tx *Transaction) completeNewTX() {
	tx.makeHash()
}

// Return ID of transaction
func (tx Transaction) GetID() []byte {
	return tx.ID
}

// returns base TX for this SQL update TX
func (tx Transaction) GetSQLBaseTX() []byte {
	return tx.SQLBaseTX
}

// IsCoinbase checks whether the transaction is currency transaction
func (tx Transaction) IsCurrencyTransfer() bool {
	if len(tx.Vin) > 0 {
		return true
	}
	return false
}

// IsCoinbase checks whether the transaction is coinbase
func (tx Transaction) IsSQLCommand() bool {
	return !tx.SQLCommand.IsEmpty()
}

// check if TX is coin base
func (tx Transaction) IsCoinbaseTransfer() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// check if needs signatures. in case of coin base doesn't need it
func (tx Transaction) NeedsSignature() bool {
	return !tx.IsCoinbaseTransfer()
}

// Hash returns the hash of the Transaction
func (tx *Transaction) makeHash() ([]byte, error) {
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

// TrimmedCopy creates a trimmed copy of Transaction to be used in signing
func (tx Transaction) Copy() (*Transaction, error) {
	//if tx.IsCoinbase() {
	//	return *tx, nil
	//}

	txCopy := &Transaction{}
	txCopy.ID = tx.ID
	txCopy.Time = tx.Time
	txCopy.Vin = tx.Vin
	txCopy.Vout = tx.Vout
	txCopy.Signature = tx.Signature
	txCopy.ByPubKey = tx.ByPubKey
	txCopy.SQLCommand = tx.SQLCommand
	txCopy.SQLBaseTX = tx.SQLBaseTX

	return txCopy, nil
}

// prepare data to sign a transaction
// this return []bytes. it is bytes representation of a TX.
func (tx *Transaction) PrepareSignData(pubKey []byte, prevTXs map[int]*Transaction) ([]byte, error) {

	pubKeyHash, _ := utils.HashPubKey(pubKey)

	// check if input transactions map is correct
	for vinInd, vin := range tx.Vin {
		if _, ok := prevTXs[vinInd]; !ok {
			return nil, errors.New("Previous transaction is not correct")
		}
		if bytes.Compare(prevTXs[vinInd].GetID(), vin.Txid) != 0 {
			return nil, errors.New("Previous transaction is not correct")
		}

		if bytes.Compare(pubKeyHash, prevTXs[vinInd].Vout[vin.Vout].PubKeyHash) != 0 {
			// check if output of previous transaction really belomgs to this pub key
			return nil, errors.New("Previous Transaction was assigned to other address")
		}
	}

	tx.ByPubKey = pubKey
	tx.Signature = []byte{}
	tx.ID = []byte{}

	return tx.ToBytes()
}

// Sets signatures for inputs. Signatures were created separately for data set prepared before
// in the function PrepareSignData
func (tx *Transaction) CompleteTransaction(signature []byte) error {
	tx.Signature = signature

	tx.completeNewTX()

	return nil
}

// Verify verifies signatures of Transaction inputs
// And total amount of inputs and outputs
func (tx *Transaction) Verify(prevTXs map[int]*Transaction) error {
	if tx.IsCoinbaseTransfer() {
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
		prevTx := prevTXs[vind]

		if prevTx.ID == nil {
			return errors.New("Previous transaction is not correct")
		}
		amount := prevTx.Vout[vin.Vout].Value
		totalinput += amount
	}
	// VERIFY signature
	// build copy to make sign data
	txCopy, err := tx.Copy()
	if err != nil {
		return err
	}
	txCopy.Signature = []byte{}
	txCopy.ID = []byte{}

	stringtosign, err := txCopy.ToBytes()

	if err != nil {
		return err
	}

	v, err := utils.VerifySignature(tx.Signature, stringtosign, tx.ByPubKey)

	if err != nil {
		return err
	}

	if !v {
		return errors.New(fmt.Sprintf("Signatire doe not match for TX %x.", tx.GetID()))
	}

	pubKeyHash, _ := utils.HashPubKey(tx.ByPubKey)

	for inID, vin := range tx.Vin {
		// full input transaction
		prevTx := prevTXs[inID]

		if bytes.Compare(prevTx.Vout[vin.Vout].PubKeyHash, pubKeyHash) != 0 {
			return errors.New(fmt.Sprintf("Sign Key Hash for input %x is different from output hash", vin.Txid))
		}
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

// Serialize returns a serialized Transaction
func (tx Transaction) serialize() ([]byte, error) {
	// to remove any references to other ponters
	// do full copy of the TX
	gob.Register(SQLUpdate{})
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		return nil, err
	}

	return encoded.Bytes(), nil
}

// DeserializeTransaction deserializes a transaction
func (tx *Transaction) DeserializeTransaction(data []byte) error {
	gob.Register(SQLUpdate{})
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(tx)

	if err != nil {
		return err
	}
	return nil
}

// converts transaction to slice of bytes
// this will be used to do a hash of transactions
func (tx Transaction) ToBytes() ([]byte, error) {
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

	err = binary.Write(buff, binary.BigEndian, tx.Signature)

	if err != nil {
		return nil, err
	}

	err = binary.Write(buff, binary.BigEndian, tx.ByPubKey)

	if err != nil {
		return nil, err
	}

	err = binary.Write(buff, binary.BigEndian, tx.SQLCommand.ToBytes())

	if err != nil {
		return nil, err
	}

	err = binary.Write(buff, binary.BigEndian, tx.SQLBaseTX)

	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func (tx Transaction) IsComplete() bool {
	if tx.ID != nil {
		return true
	}
	return false
}

// UsesKey checks whether the address initiated the transaction

func (tx Transaction) CreatedByPubKeyHash(pubKeyHash []byte) bool {
	lockingHash, _ := utils.HashPubKey(tx.ByPubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

//
func (tx Transaction) GetTime() int64 {
	return tx.Time
}

//
func (tx *Transaction) SetSQLPart(sql SQLUpdate) {
	tx.SQLCommand = sql
}

// returns SQL command as string
func (tx Transaction) GetSQLQuery() string {
	if len(tx.SQLCommand.Query) > 0 {
		return string(tx.SQLCommand.Query)
	}
	return ""
}

// String returns a human-readable representation of a transaction
func (tx Transaction) String() string {
	var lines []string
	from := "Coin Base"
	fromhash := []byte{}

	if !tx.IsCoinbaseTransfer() {
		from, _ = utils.PubKeyToAddres(tx.ByPubKey)
		fromhash, _ = utils.HashPubKey(tx.ByPubKey)
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

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	if amount > 0 {
		lines = append(lines, fmt.Sprintf("    FROM %s TO %s VALUE %f", from, to, amount))
	}
	lines = append(lines, fmt.Sprintf("    Time %d (%s)", tx.Time, time.Unix(0, tx.Time)))

	if !tx.IsCoinbaseTransfer() && !tx.IsSQLCommand() {
		for i, input := range tx.Vin {
			lines = append(lines, fmt.Sprintf("     Input %d:", i))
			lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
			lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
		}
	}

	for i, output := range tx.Vout {
		address, _ := utils.PubKeyHashToAddres(output.PubKeyHash)
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %f", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
		lines = append(lines, fmt.Sprintf("       Address: %s", address))
	}

	if tx.IsSQLCommand() {
		lines = append(lines, fmt.Sprintf("    SQL: %s", tx.GetSQLQuery()))
		lines = append(lines, fmt.Sprintf("    By: %s", from))
		lines = append(lines, fmt.Sprintf("    Based On: %x", tx.SQLBaseTX))
	}

	return strings.Join(lines, "\n")
}
