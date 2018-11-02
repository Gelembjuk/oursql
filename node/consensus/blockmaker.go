package consensus

import (
	"errors"
	"fmt"
	"time"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/blockchain"
	"github.com/gelembjuk/oursql/node/config"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
	"github.com/gelembjuk/oursql/node/transactions"
)

type NodeBlockMaker struct {
	DB            database.DBManager
	Logger        *utils.LoggerMan
	MinterAddress string // this is the wallet that will receive for mining
	PreparedBlock *structures.Block
	config        *ConsensusConfig
}

func (n *NodeBlockMaker) SetDBManager(DB database.DBManager) {
	n.DB = DB
}
func (n *NodeBlockMaker) SetLogManager(Logger *utils.LoggerMan) {
	n.Logger = Logger
}
func (n *NodeBlockMaker) SetMinterAddress(minter string) {
	n.MinterAddress = minter
}

func (n *NodeBlockMaker) PrepareNewBlock() (int, error) {

	if n.PreparedBlock != nil {
		return BlockPrepare_Error, errors.New("There is a block prepared already")
	}

	if !n.checkGoodTimeToMakeBlock() {
		return BlockPrepare_NotGoodTime, nil
	}

	if !n.checkUnapprovedCache() {
		return BlockPrepare_NoTransactions, nil
	}

	n.PreparedBlock = nil

	err := n.doPrepareNewBlock()

	if err != nil {
		return BlockPrepare_Error, err
	}

	if n.PreparedBlock == nil {
		return BlockPrepare_NoTransactions, nil
	}

	return BlockPrepare_Done, nil
}

// check if a block was prepared already
func (n *NodeBlockMaker) IsBlockPrepared() bool {
	return n.PreparedBlock != nil
}

// Transaction operations and cache manager
func (n *NodeBlockMaker) getTransactionsManager() transactions.TransactionsManagerInterface {
	return transactions.NewManager(n.DB, n.Logger, n.config.GetInfoForTransactions())
}

// Blockchain DB manager.
func (n *NodeBlockMaker) getBlockchainManager() *blockchain.Blockchain {
	bcm, _ := blockchain.NewBlockchainManager(n.DB, n.Logger)

	return bcm
}

// Checks if this is good time for this node to make a block
// In this version it is always true

func (n *NodeBlockMaker) checkGoodTimeToMakeBlock() bool {
	return true
}

// Check if there are abough unapproved transactions to make a block
func (n *NodeBlockMaker) checkUnapprovedCache() bool {
	count, err := n.getTransactionsManager().GetUnapprovedCount()

	if err != nil {
		n.Logger.Trace.Printf("Error when check unapproved cache: %s", err.Error())
		return false
	}

	n.Logger.Trace.Printf("Transaction in cache - %d", count)

	min, max, err := n.getTransactionNumbersLimits(nil)

	if count >= min {
		if count > max {
			count = max
		}
		return true
	}
	return false
}
func (n *NodeBlockMaker) SetPreparedBlock(block *structures.Block) error {
	n.PreparedBlock = block
	return nil
}

// Makes new block, without a hash. Only finds transactions to add to a block

func (n *NodeBlockMaker) doPrepareNewBlock() error {
	// firstly, check count of transactions to know if there are enough
	count, err := n.getTransactionsManager().GetUnapprovedCount()

	if err != nil {
		return err
	}
	min, max, err := n.getTransactionNumbersLimits(nil)

	if err != nil {
		return err
	}

	n.Logger.Trace.Printf("Minting: Found %d transaction from minimum %d\n", count, min)

	if count >= min {
		// number of transactions is fine
		if count > max {
			count = max
		}
		// get unapproved transactions
		txs, err := n.getTransactionsManager().GetUnapprovedTransactionsForNewBlock(count)

		if err != nil {
			return err
		}

		if len(txs) < min {
			return errors.New("No enought valid transactions! Waiting for new ones...")
		}

		n.Logger.Trace.Printf("Minting: All good. New block assigned to address %s\n", n.MinterAddress)

		newBlock, err := n.makeNewBlockFromTransactions(txs)

		if err != nil {
			return err
		}

		n.Logger.Trace.Printf("Minting: New block Prepared. Not yet complete\n")

		n.PreparedBlock = newBlock

	}

	return nil
}

// finalise a block. in this place we do MIMING
// Block was prepared. Now do final work for this block. In our case it is PoW process
func (n *NodeBlockMaker) CompleteBlock() (*structures.Block, error) {
	if n.PreparedBlock == nil {
		return nil, errors.New("Block was not prepared")
	}
	b := n.PreparedBlock
	// NOTE
	// We don't check if transactions are valid in this place .
	// we did checks before in the calling function
	// we checked each tranaction if it has correct signature,
	// it inputs  are not yet stent before
	// if there is no 2 transaction with same input in one block

	n.Logger.Trace.Printf("Minting: Start proof of work for the block\n")

	starttime := time.Now()
	pow := NewProofOfWork(b, n.config.Settings)

	nonce, hash, err := pow.Run()

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Pow error: %s", err))
	}

	b.Hash = hash[:]
	b.Nonce = nonce

	if config.MinimumBlockBuildingTime > 0 {
		for t := time.Since(starttime).Seconds(); t < float64(config.MinimumBlockBuildingTime); t = time.Since(starttime).Seconds() {
			time.Sleep(1 * time.Second)
			n.Logger.Trace.Printf("Sleep")
		}
	}

	n.Logger.Trace.Printf("Minting: New hash is %x\n", b.Hash)

	return b, nil
}

// this builds a block object from given transactions list
// adds coinbase transacion (prize for miner)
func (n *NodeBlockMaker) makeNewBlockFromTransactions(transactions []structures.Transaction) (*structures.Block, error) {
	// get last block info
	lastHash, lastHeight, err := n.getBlockchainManager().GetState()

	if err != nil {
		return nil, err
	}

	// add transaction - prize for miner
	cbTx, errc := structures.NewCoinbaseTransaction(n.MinterAddress, "", n.config.CoinsForBlockMade)

	if errc != nil {
		return nil, errc
	}

	transactions = append(transactions, *cbTx)
	/*
		txlist := []*transaction.Transaction{}

		for _, t := range transactions {
			tx, _ := t.Copy()
			txlist = append(txlist, &tx)
		}
	*/
	newblock := structures.Block{}
	err = newblock.PrepareNewBlock(transactions, lastHash[:], lastHeight+1)

	if err != nil {
		return nil, err
	}

	return &newblock, nil
}

// Verify the block. We check if it is correct agains previous block
// Verify a block against blockchain
// RULES
// 0. Verification is done agains blockchain branch starting from prevblock, not current top branch
// 1. There can be only 1 transaction make reward per block
// 2. number of transactions must be in correct ranges (reward transaction is not calculated)
// 3. transactions can have as input other transaction from this block and it must be listed BEFORE
//   (output must be before input in same block)
// 4. all inputs must be in blockchain (correct unspent inputs)
// 5. Additionally verify each transaction agains signatures, total amount, balance etc
// 6. Verify hash is correc agains rules
func (n *NodeBlockMaker) VerifyBlock(block *structures.Block) error {
	//6. Verify hash
	pow := NewProofOfWork(block, n.config.Settings)

	valid, err := pow.Validate()

	if err != nil {
		return err
	}

	if !valid {
		return errors.New("Block hash is not valid")
	}
	n.Logger.Trace.Println("block hash verified")
	// 2. check number of TX
	txnum := len(block.Transactions) - 1 /*minus coinbase TX*/

	min, max, err := n.getTransactionNumbersLimits(block)

	if err != nil {
		return err
	}

	if txnum < min {
		return errors.New("Number of transactions is too low")
	}

	if txnum > max {
		return errors.New("Number of transactions is too high")
	}

	// 1
	coinbaseused := false

	prevTXs := []structures.Transaction{}

	for _, tx := range block.Transactions {
		if tx.IsCoinbaseTransfer() {
			if coinbaseused {
				return errors.New("2 coin base TX in the block")
			}
			coinbaseused = true
		}
		vtx, err := n.getTransactionsManager().VerifyTransaction(&tx, prevTXs, block.PrevBlockHash)

		if err != nil {
			return errors.New(fmt.Sprintf("TX verify during block verify. Error: %s", err.Error()))
		}

		if !vtx {
			return errors.New(fmt.Sprintf("Transaction in a block is not valid: %x", tx.GetID()))
		}
		//n.Logger.Trace.Printf("checked %x . add it to previous list", tx.GetID())
		prevTXs = append(prevTXs, tx)
	}
	// 1.
	if !coinbaseused {
		return errors.New("No coinbase TX in the block")
	}
	return nil
}

//Get minimum and maximum number of transaction allowed in block for current chain
func (n *NodeBlockMaker) getTransactionNumbersLimits(block *structures.Block) (int, int, error) {
	var h int

	if block == nil {
		bcm := n.getBlockchainManager()

		bestHeight, err := bcm.GetBestHeight()

		if err != nil {
			return 0, 0, err
		}
		h = bestHeight + 1
	} else {
		h = block.Height
	}
	pow := NewProofOfWork(nil, n.config.Settings)

	min, max := pow.GetTransactionLimitsPerBlock(h)

	n.Logger.Trace.Printf("TX count limits %d - %d", min, max)
	return min, max, nil
}
