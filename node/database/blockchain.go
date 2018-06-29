package database

import (
	"bytes"
)

const blocksTable = "blocks"
const blockChainTable = "blockchain"

type Blockchain struct {
	DB              *MySQLDB
	blocksTable     string
	blockChainTable string
}

func (bc *Blockchain) getBlocksTable() string {
	if bc.blocksTable == "" {
		bc.blocksTable = bc.DB.tablesPrefix + blocksTable
	}
	return bc.blocksTable
}

func (bc *Blockchain) getBlockChainTable() string {
	if bc.blockChainTable == "" {
		bc.blockChainTable = bc.DB.tablesPrefix + blockChainTable
	}
	return bc.blockChainTable
}

// create bucket etc. DB is already inited
func (bc *Blockchain) InitDB() error {
	err := bc.DB.CreateTable(bc.getBlocksTable(), "VARBINARY(100)", "LONGBLOB")

	if err != nil {
		return err
	}

	return bc.DB.CreateTable(bc.getBlockChainTable(), "VARBINARY(100)", "VARBINARY(300)")
}

// Get block on the top of blockchain
func (bc *Blockchain) GetTopBlock() ([]byte, error) {
	topHash, err := bc.GetTopHash()

	if err != nil {
		return nil, err
	}

	return bc.GetBlock(topHash)
}

//  Check if block exists by hash
func (bc *Blockchain) CheckBlockExists(hash []byte) (bool, error) {
	// we just load this bloc data from db .
	blockData, err := bc.GetBlock(hash)

	if err != nil {
		return false, err
	}

	if len(blockData) > 0 {
		return true, nil
	}
	return false, nil
}

// Add block to the top of block chain
func (bc *Blockchain) PutBlockOnTop(hash []byte, blockdata []byte) error {
	err := bc.PutBlock(hash, blockdata)

	if err != nil {
		return err
	}

	return bc.SaveTopHash(hash)
}

// Get block data by hash. It returns just []byte and and must be deserialised on ther place
func (bc *Blockchain) GetBlock(hash []byte) ([]byte, error) {
	return bc.DB.Get(bc.getBlocksTable(), hash)
}

// Add block record
func (bc *Blockchain) PutBlock(hash []byte, blockdata []byte) error {
	return bc.DB.Put(bc.getBlocksTable(), hash, blockdata)
}

// Delete block record
func (bc *Blockchain) DeleteBlock(hash []byte) error {
	return bc.DB.Delete(bc.getBlocksTable(), hash)
}

// Save top level block hash
func (bc *Blockchain) SaveTopHash(hash []byte) error {
	return bc.DB.Put(bc.getBlocksTable(), []byte("l"), hash)
}

// Get top level block hash
func (bc *Blockchain) GetTopHash() ([]byte, error) {
	h, err := bc.DB.Get(bc.getBlocksTable(), []byte("l"))

	if err != nil {
		return nil, err
	}
	if h == nil {
		return nil, NewNotFoundDBError("tophash")
	}
	return h, nil
}

// Save first (or genesis) block hash. It should be called when blockchain is created
func (bc *Blockchain) SaveFirstHash(hash []byte) error {
	return bc.DB.Put(bc.getBlocksTable(), []byte("f"), hash)
}

func (bc *Blockchain) GetFirstHash() ([]byte, error) {
	h, err := bc.DB.Get(bc.getBlocksTable(), []byte("f"))

	if err != nil {
		return nil, err
	}
	if h == nil {
		return nil, NewNotFoundDBError("firsthash")
	}
	return h, nil
}

// Read record from chain table.
func (bc *Blockchain) getRecordFromChain(hash []byte) []byte {
	b, _ := bc.DB.Get(bc.getBlockChainTable(), hash)
	return b
}

// Save record to chain table
func (bc *Blockchain) saveRecordToChain(hash []byte, hashData []byte) error {
	return bc.DB.Put(bc.getBlockChainTable(), hash, hashData)
}

// Delete record from chain table
func (bc *Blockchain) deleteRecordFromChain(hash []byte) error {
	return bc.DB.Delete(bc.getBlockChainTable(), hash)
}

// add block to chain
func (bc *Blockchain) AddToChain(hash, prevHash []byte) error {
	length := len(hash)

	if length == 0 {
		return NewHashEmptyDBError()
	}

	emptyHash := make([]byte, length)

	// maybe it already exists in chain. check it
	// TODO . not sure if we need to do this check. ignore for now

	hashBytes := make([]byte, length*2)

	if len(prevHash) > 0 {
		// get prev hash and put this hash as next

		prRec := bc.getRecordFromChain(prevHash)

		if len(prRec) < length*2 {
			return NewHashNotFoundDBError("Previous hash is not found in the chain")
		}

		exNext := make([]byte, length)

		copy(exNext, prRec[length:])

		if bytes.Compare(exNext, emptyHash) > 0 {
			return NewHashDBError("Previous hash already has a next hash")
		}

		copy(prRec[length:], hash)

		err := bc.saveRecordToChain(prevHash, prRec)

		if err != nil {
			return err
		}

		copy(hashBytes[0:], prevHash)
	}

	return bc.saveRecordToChain(hash, hashBytes)

}

// remove block from chain
func (bc *Blockchain) RemoveFromChain(hash []byte) error {
	length := len(hash)

	if length == 0 {
		return NewHashEmptyDBError()
	}

	emptyHash := make([]byte, length)

	// get prev hash and put this hash as next
	hashBytes := bc.getRecordFromChain(hash)

	if len(hashBytes) < length*2 {
		return NewHashNotFoundDBError(" ")
	}

	nextHash := make([]byte, length)

	copy(nextHash, hashBytes[length:])

	if bytes.Compare(nextHash, emptyHash) > 0 {
		return NewHashDBError("Only last hash can be removed")
	}

	prevHash := make([]byte, length)
	copy(prevHash, hashBytes[0:length])

	if bytes.Compare(prevHash, emptyHash) > 0 {

		prevHashBytes := bc.getRecordFromChain(prevHash)

		if len(prevHashBytes) < length*2 {

			return NewHashNotFoundDBError("Previous hash is not found")
		}
		copy(prevHashBytes[length:], emptyHash)

		err := bc.saveRecordToChain(prevHash, prevHashBytes)

		if err != nil {
			return err
		}

	}

	return bc.deleteRecordFromChain(hash)
}

func (bc *Blockchain) BlockInChain(hash []byte) (bool, error) {
	length := len(hash)

	if length == 0 {
		return false, NewHashEmptyDBError()
	}

	h := bc.getRecordFromChain(hash)

	if len(h) == length*2 {
		return true, nil
	}

	return false, nil
}

func (bc *Blockchain) GetLocationInChain(hash []byte) (bool, []byte, []byte, error) {
	length := len(hash)

	if length == 0 {
		return false, nil, nil, NewHashEmptyDBError()
	}
	var prevHash []byte
	var nextHash []byte

	emptyHash := make([]byte, length)

	h := bc.getRecordFromChain(hash)

	if len(h) == len(hash)*2 {

		prevHash = make([]byte, length)
		nextHash = make([]byte, length)
		copy(prevHash, h[:length])
		copy(nextHash, h[length:])

		if bytes.Compare(prevHash, emptyHash) == 0 {
			prevHash = []byte{}
		}

		if bytes.Compare(nextHash, emptyHash) == 0 {
			nextHash = []byte{}
		}

		return true, prevHash, nextHash, nil
	}

	return false, prevHash, nextHash, nil
}
