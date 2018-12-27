package structures

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"time"

	"github.com/gelembjuk/oursql/lib/utils"
)

// Block represents a block in the blockchain
type Block struct {
	Timestamp     int64
	Transactions  []Transaction
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
	Height        int
}

// short info about a block. to exchange over network
type BlockShort struct {
	PrevBlockHash []byte
	Hash          []byte
	Height        int
}

// simpler representation of a block. transactions are presented as strings
type BlockSimpler struct {
	Timestamp     int64
	Transactions  []string
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
	Height        int
}

// Reverce list of blocks
func ReverseBlocksSlice(ss []*Block) {
	last := len(ss) - 1

	if last < 1 {
		return
	}

	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}

// Reverce list of blocks for short info
func ReverseBlocksShortSlice(ss []*BlockShort) {
	last := len(ss) - 1

	if last < 1 {
		return
	}

	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}

}

// Serialise BlockShort to bytes
func (b *BlockShort) Serialize() ([]byte, error) {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}

// Deserialize BlockShort from bytes
func (b *BlockShort) DeserializeBlock(d []byte) error {

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&b)

	if err != nil {
		return err
	}

	return nil
}

// Returns short copy of a block. It is just hash + prevhash
func (b *Block) GetShortCopy() *BlockShort {
	bs := BlockShort{}
	bs.Hash = b.Hash[:]
	bs.PrevBlockHash = b.PrevBlockHash[:]
	bs.Height = b.Height

	return &bs
}

// Returns simpler copy of a block. This is the version for easy print
// TODO . not sure we really need this
func (b *Block) GetSimpler() *BlockSimpler {
	Block := BlockSimpler{}
	Block.Hash = b.Hash[:]
	Block.Height = b.Height
	Block.PrevBlockHash = b.PrevBlockHash[:]

	Block.Transactions = []string{}

	for _, tx := range b.Transactions {
		Block.Transactions = append(Block.Transactions, tx.String())
	}
	return &Block
}

// Creates copy of a block
func (b *Block) Copy() *Block {
	bc := Block{}
	bc.Timestamp = b.Timestamp
	bc.Transactions = []Transaction{}

	bc.PrevBlockHash = make([]byte, len(b.PrevBlockHash))

	if len(b.PrevBlockHash) > 0 {
		copy(bc.PrevBlockHash, b.PrevBlockHash)
	}

	bc.Hash = make([]byte, len(b.Hash))

	if len(b.Hash) > 0 {
		copy(bc.Hash, b.Hash)
	}

	bc.Nonce = b.Nonce
	bc.Height = b.Height

	for _, t := range b.Transactions {
		tc, _ := t.Copy()
		bc.Transactions = append(bc.Transactions, *tc)
	}
	return &bc
}

// Fills a block with transactions. But without signatures
func (b *Block) PrepareNewBlock(transactions []Transaction, prevBlockHash []byte, height int) error {
	b.Timestamp = time.Now().Unix()
	b.Transactions = []Transaction{}

	for _, tx := range transactions {
		b.Transactions = append(b.Transactions, tx)
	}

	b.PrevBlockHash = make([]byte, len(prevBlockHash))

	if len(prevBlockHash) > 0 {
		copy(b.PrevBlockHash, prevBlockHash)
	}

	b.Hash = []byte{}
	b.Nonce = 0
	b.Height = height

	return nil
}

// HashTransactions returns a hash of the transactions in the block
func (b *Block) HashTransactions() ([]byte, error) {
	var transactions [][]byte

	for _, tx := range b.Transactions {
		txser, err := tx.ToBytes()

		if err != nil {

			return nil, err
		}
		transactions = append(transactions, txser)
	}

	mTree := utils.NewMerkleTree(transactions)

	return mTree.RootNode.Data, nil
}

// Serialize serializes the block
func (b *Block) Serialize() ([]byte, error) {
	var result bytes.Buffer

	gob.Register(&Transaction{})

	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Block serialise error %s", err.Error()))
	}

	return result.Bytes(), nil
}

// DeserializeBlock deserializes a block
func (b *Block) DeserializeBlock(d []byte) error {
	gob.Register(&Transaction{})
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&b)

	if err != nil {
		return err
	}

	return nil
}

func (b BlockSimpler) String() string {
	s := fmt.Sprintf("============ Block %x ============\n", b.Hash)
	s = s + fmt.Sprintf("Height: %d\n", b.Height)
	s = s + fmt.Sprintf("Prev. block: %x\n", b.PrevBlockHash)

	for _, tx := range b.Transactions {
		s = s + fmt.Sprintln(tx)
	}
	s = s + fmt.Sprintf("\n")

	return s
}
