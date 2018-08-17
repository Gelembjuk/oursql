package transactions

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/blockchain"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
)

type transactionsIndex struct {
	DB     database.DBManager
	Logger *utils.LoggerMan
}

type TransactionsIndexSpentOutputs struct {
	OutInd      int
	TXWhereUsed []byte
	InInd       int
	BlockHash   []byte
}

func newTransactionIndex(DB database.DBManager, Logger *utils.LoggerMan) *transactionsIndex {
	return &transactionsIndex{DB, Logger}
}

func (tiso TransactionsIndexSpentOutputs) String() string {
	return fmt.Sprintf("OI %d used in %x II %d block %x", tiso.OutInd, tiso.TXWhereUsed, tiso.InInd, tiso.BlockHash)
}
func (ti *transactionsIndex) BlocksAdded(blocks []*structures.Block) error {
	for _, block := range blocks {

		err := ti.BlockAdded(block)

		if err != nil {

			return err
		}
	}
	return nil
}

// Block added. We need to update index of transactions
func (ti *transactionsIndex) BlockAdded(block *structures.Block) error {
	txdb, err := ti.DB.GetTransactionsObject()

	if err != nil {
		return err
	}

	for _, tx := range block.Transactions {
		// get current lit of blocks
		blocksHashes, err := txdb.GetBlockHashForTX(tx.GetID())

		if err != nil {
			return err
		}

		hashes := [][]byte{}

		if blocksHashes != nil {
			hashes, err = ti.DeserializeHashes(blocksHashes)

			if err != nil {
				return err
			}
		}
		hashes = append(hashes, block.Hash[:])
		ti.Logger.Trace.Printf("block add. %x new list %x %d", tx.GetID(), hashes, len(hashes))
		blocksHashes, err = ti.SerializeHashes(hashes)

		if err != nil {
			return err
		}

		err = txdb.PutTXToBlockLink(tx.GetID(), blocksHashes)

		if err != nil {
			return err
		}
		if tx.IsCoinbaseTransfer() {
			continue
		}
		currTx := tx

		// for each input we save list of tranactions where iput was used
		for inInd, vin := range currTx.Vin {
			// get existing ecordsfor this input
			to, err := txdb.GetTXSpentOutputs(vin.Txid)

			if err != nil {
				return err
			}

			outs := []TransactionsIndexSpentOutputs{}

			if to != nil {

				var err error
				outs, err = ti.DeserializeOutputs(to)

				if err != nil {
					return err
				}
			}

			outs = append(outs, TransactionsIndexSpentOutputs{vin.Vout, currTx.GetID(), inInd, block.Hash[:]})

			to, err = ti.SerializeOutputs(outs)

			if err != nil {
				return err
			}
			// by this ID we can know which output were already used and in which transactions
			err = txdb.PutTXSpentOutputs(vin.Txid, to)

			if err != nil {
				return err
			}
		}
	}
	return nil
}
func (ti *transactionsIndex) BlocksRemoved(blocks []*structures.Block) error {
	for _, block := range blocks {

		err := ti.BlockRemoved(block)

		if err != nil {

			return err
		}
	}
	return nil
}
func (ti *transactionsIndex) BlockRemoved(block *structures.Block) error {
	txdb, err := ti.DB.GetTransactionsObject()

	if err != nil {
		return err
	}

	for _, tx := range block.Transactions {
		blocksHashes, err := txdb.GetBlockHashForTX(tx.GetID())

		if err != nil {
			return err
		}

		hashes := [][]byte{}

		if blocksHashes != nil {
			hashes, err = ti.DeserializeHashes(blocksHashes)

			if err != nil {
				return err
			}
		}
		ti.Logger.Trace.Printf("block remove. before %s", hashes)
		newHashes := [][]byte{}

		for _, hash := range hashes {
			if bytes.Compare(hash, block.Hash) != 0 {
				newHashes = append(newHashes, hash)
			}
		}
		ti.Logger.Trace.Printf("block remove. after %s", newHashes)
		if len(newHashes) > 0 {
			blocksHashes, err = ti.SerializeHashes(newHashes)

			if err != nil {
				return err
			}

			err = txdb.PutTXToBlockLink(tx.GetID(), blocksHashes)

			if err != nil {
				return err
			}
		} else {
			txdb.DeleteTXToBlockLink(tx.GetID())
		}
		if tx.IsCoinbaseTransfer() {
			continue
		}

		// remove inputs from used outputs
		for _, vin := range tx.Vin {
			// get existing ecordsfor this input
			to, err := txdb.GetTXSpentOutputs(vin.Txid)

			if err != nil {
				return err
			}

			if to == nil {
				continue
			}
			outs, err := ti.DeserializeOutputs(to)

			if err != nil {
				return err
			}
			newOoutputs := []TransactionsIndexSpentOutputs{}

			for _, o := range outs {
				if o.OutInd != vin.Vout {
					newOoutputs = append(newOoutputs, o)
				}
			}
			outs = newOoutputs[:]

			if len(outs) > 0 {
				to, err = ti.SerializeOutputs(outs)

				if err != nil {
					return err
				}
				// by this ID we can know which output were already used and in which transactions
				txdb.PutTXSpentOutputs(vin.Txid, to)
			} else {
				txdb.DeleteTXSpentData(vin.Txid)
			}

		}
	}
	return nil
}

// Reindex cach of trsnactions pointers to block
func (ti *transactionsIndex) Reindex() error {
	ti.Logger.Trace.Println("TXCache.Reindex: Prepare to recreate bucket")

	txdb, err := ti.DB.GetTransactionsObject()

	if err != nil {
		return err
	}

	err = txdb.TruncateDB()

	if err != nil {
		return err
	}

	ti.Logger.Trace.Println("TXCache.Reindex: Bucket created")

	bci, err := blockchain.NewBlockchainIterator(ti.DB)

	if err != nil {
		return err
	}

	for {
		block, err := bci.Next()

		if err != nil {

			return err
		}

		ti.Logger.Trace.Printf("TXCache.Reindex: Process block: %d, %x", block.Height, block.Hash)

		err = ti.BlockAdded(block)

		if err != nil {
			return err
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	ti.Logger.Trace.Println("TXCache.Reindex: Done")
	return nil
}

// Serialize. We need this to store data in DB in bytes

func (ti *transactionsIndex) SerializeOutputs(outs []TransactionsIndexSpentOutputs) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

// Deserialize data from bytes loaded fom DB

func (ti *transactionsIndex) DeserializeOutputs(data []byte) ([]TransactionsIndexSpentOutputs, error) {
	var outputs []TransactionsIndexSpentOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	if err != nil {
		return nil, err
	}

	return outputs, nil
}

// Serialise list of hashes
func (ti *transactionsIndex) SerializeHashes(hashes [][]byte) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(hashes)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

// Deserialize list of hashes

func (ti *transactionsIndex) DeserializeHashes(data []byte) ([][]byte, error) {
	var hashes [][]byte

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&hashes)
	if err != nil {
		return nil, err
	}

	return hashes, nil
}

func (ti *transactionsIndex) GetTranactionBlocks(txID []byte) ([][]byte, error) {
	txdb, err := ti.DB.GetTransactionsObject()

	if err != nil {
		return nil, err
	}

	blockHashBytes, err := txdb.GetBlockHashForTX(txID)

	if err != nil {
		ti.Logger.Trace.Printf("GetTranactionBlocks Error 1: %s", err.Error())
		return nil, err
	}

	if blockHashBytes == nil {
		return [][]byte{}, nil
	}

	hashes, err := ti.DeserializeHashes(blockHashBytes)

	if err != nil {
		ti.Logger.Trace.Printf("GetTranactionBlocks Error 2: %s", err.Error())
		return nil, err
	}

	return hashes, nil
}

// Get list of spent outputs for TX
// It uses the block and top block hashes to find a range of blocks where a spending can be
// This index can contains spending in some other parallel branches. We use top and bottom hashes
// to set a chain where to look for spendings
func (ti *transactionsIndex) GetTranactionOutputsSpent(txID []byte, blockHash []byte, topHash []byte) ([]TransactionsIndexSpentOutputs, error) {
	txdb, err := ti.DB.GetTransactionsObject()

	if err != nil {
		return nil, err
	}

	to, err := txdb.GetTXSpentOutputs(txID)

	if err != nil {
		return nil, err
	}

	res := []TransactionsIndexSpentOutputs{}

	if to != nil {
		tmpres, err := ti.DeserializeOutputs(to)

		if err != nil {
			return nil, err
		}

		// filter this list by block hashes
		res, err = ti.filterTranactionOutputsSpent(tmpres, blockHash, topHash)

		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (ti *transactionsIndex) filterTranactionOutputsSpent(outPuts []TransactionsIndexSpentOutputs,
	blockHash []byte, topHash []byte) ([]TransactionsIndexSpentOutputs, error) {

	bcMan, err := blockchain.NewBlockchainManager(ti.DB, ti.Logger)

	if err != nil {
		return nil, err
	}

	res := []TransactionsIndexSpentOutputs{}

	for _, o := range outPuts {

		present, err := bcMan.CheckBlockIsInRange(o.BlockHash, blockHash, topHash)

		if err != nil {
			return nil, err
		}

		if present {
			res = append(res, o)
		}
	}

	return res, nil
}

// Get full TX, spending status and block hash for TX by ID
func (ti *transactionsIndex) GetCurrencyTransactionAllInfo(txID []byte, topHash []byte) (*structures.Transaction, []TransactionsIndexSpentOutputs, []byte, error) {
	localError := func(err error) (*structures.Transaction, []TransactionsIndexSpentOutputs, []byte, error) {
		return nil, nil, nil, err
	}

	blockHashes, err := ti.GetTranactionBlocks(txID)

	if err != nil {
		return localError(err)
	}

	bcMan, err := blockchain.NewBlockchainManager(ti.DB, ti.Logger)

	if err != nil {
		return localError(err)
	}

	// find which of hashes corresponds to provied top
	blockHash, err := bcMan.ChooseHashUnderTip(blockHashes, topHash)

	if err != nil {
		return localError(err)
	}

	if blockHash == nil {
		return localError(nil)
	}

	spentOuts, err := ti.GetTranactionOutputsSpent(txID, blockHash, topHash)

	if err != nil {
		return localError(err)
	}

	tx, err := bcMan.GetTransactionFromBlock(txID, blockHash)

	if err != nil {
		return localError(err)
	}

	return tx, spentOuts, blockHash, nil
}

// Get TX object from BC under given topHash
func (ti *transactionsIndex) GetTransaction(txID []byte, topHash []byte) (*structures.Transaction, error) {
	localError := func(err error) (*structures.Transaction, error) {
		return nil, err
	}

	blockHashes, err := ti.GetTranactionBlocks(txID)

	if err != nil {
		return localError(err)
	}

	bcMan, err := blockchain.NewBlockchainManager(ti.DB, ti.Logger)

	if err != nil {
		return localError(err)
	}

	// find which of hashes corresponds to provied top
	blockHash, err := bcMan.ChooseHashUnderTip(blockHashes, topHash)

	if err != nil {
		return localError(err)
	}

	if blockHash == nil {
		return localError(nil)
	}

	tx, err := bcMan.GetTransactionFromBlock(txID, blockHash)

	if err != nil {
		return localError(err)
	}

	return tx, nil
}
