package structures

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/gelembjuk/oursql/lib"
)

// return BlockShort object from bytes
func NewBlockShortFromBytes(bsdata []byte) (*BlockShort, error) {
	bs := &BlockShort{}
	err := bs.DeserializeBlock(bsdata)

	if err != nil {
		return nil, err
	}
	return bs, nil
}

// Make Block object from bytes
func NewBlockFromBytes(bsdata []byte) (*Block, error) {
	bs := &Block{}
	err := bs.DeserializeBlock(bsdata)

	if err != nil {
		return nil, err
	}
	return bs, nil
}

// New "currency" transaction.
func NewCurrencyTransaction(inputs []TXInput, outputs []TXOutput) (TransactionInterface, error) {
	return &CurrencyTransaction{nil, inputs, outputs, 0}, nil
}

// Serialize Transaction
func SerializeTransaction(tx TransactionInterface) ([]byte, error) {
	// add TX type to know how to deSerialize
	var txType byte
	var txData []byte
	var err error

	if tx.CheckTypeIs(TXTypeCurrency) {
		txType = txTypeCurrency

		txC := tx.(*CurrencyTransaction)

		txData, err = txC.serialize()

		if err != nil {
			return nil, err
		}
	} else if tx.CheckTypeIs(TXTypeSQL) {
		txType = txTypeSQL
		return nil, errors.New("Not implemented yet")
	} else {
		return nil, errors.New("Unknown type")
	}

	return append([]byte{txType}, txData...), nil
}

// Serialize Transaction
func DeserializeTransaction(txData []byte) (TransactionInterface, error) {
	// get type from first byte
	txType := txData[0]
	txData = txData[1:]

	if txType == txTypeCurrency {
		tx := &CurrencyTransaction{}
		err := tx.DeserializeTransaction(txData)

		if err != nil {
			return nil, err
		}
		return tx, nil
	}
	return nil, errors.New("Unknown TX type")
}

// New "currency" Coin Base transaction. This transaction must be present in each new block
func NewCurrencyCoinbaseTransaction(to, data string) (TransactionInterface, error) {
	if data == "" {
		randData := make([]byte, 20)
		_, err := rand.Read(randData)

		if err != nil {
			return nil, err
		}

		data = fmt.Sprintf("%x", randData)
	}
	tx := &CurrencyTransaction{}
	txin := TXInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTXOutput(lib.CurrencyPaymentForBlockMade, to)
	tx.Vin = []TXInput{txin}
	tx.Vout = []TXOutput{*txout}

	tx.Hash()

	return tx, nil
}

// Sorting of transactions slice
type Transactions []TransactionInterface

func (c Transactions) Len() int           { return len(c) }
func (c Transactions) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c Transactions) Less(i, j int) bool { return c[i].GetTime() < c[j].GetTime() }
