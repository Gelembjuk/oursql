package consensus

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/gelembjuk/oursql/lib"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/blockchain"
	"github.com/gelembjuk/oursql/node/config"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/dbquery"
	"github.com/gelembjuk/oursql/node/structures"
	"github.com/gelembjuk/oursql/node/transactions"
)

type NodeBlockMaker struct {
	DB            database.DBManager
	Logger        *utils.LoggerMan
	MinterAddress string // this is the wallet that will receive for mining
	PreparedBlock *structures.Block
	config        *ConsensusConfig
	verifyMan     *verifyManager
}

func (n NodeBlockMaker) getQueryParser() dbquery.QueryProcessorInterface {
	return dbquery.NewQueryProcessor(n.DB, n.Logger)
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

// Transaction operations and cache manager
func (n *NodeBlockMaker) getTransactionsManager() transactions.TransactionsManagerInterface {
	return transactions.NewManager(n.DB, n.Logger, n.config.GetInfoForTransactions())
}

// Blockchain DB manager.
func (n *NodeBlockMaker) getBlockchainManager() *blockchain.Blockchain {
	bcm, _ := blockchain.NewBlockchainManager(n.DB, n.Logger)

	return bcm
}
func (n NodeBlockMaker) getVerifyManager(prevBlockNumber int) *verifyManager {
	if n.verifyMan != nil {
		n.verifyMan.previousBlockHeigh = prevBlockNumber
		return n.verifyMan
	}
	n.verifyMan = &verifyManager{}
	n.verifyMan.config = n.config
	n.verifyMan.logger = n.Logger
	n.verifyMan.previousBlockHeigh = prevBlockNumber
	return n.verifyMan
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

	//n.Logger.Trace.Printf("Transaction in cache - %d", count)

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

// Returns IDs of transactions selected to do a next block
func (n *NodeBlockMaker) GetPreparedBlockTransactionsIDs() ([][]byte, error) {
	if n.PreparedBlock == nil {
		return nil, errors.New("Block was not prepared")
	}
	list := [][]byte{}
	for _, tx := range n.PreparedBlock.Transactions {
		if tx.IsCoinbaseTransfer() {
			continue
		}
		list = append(list, tx.GetID())
	}
	return list, nil
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
			//n.Logger.Trace.Printf("Sleep")
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
func (n *NodeBlockMaker) VerifyBlock(block *structures.Block, flags int) error {
	//6. Verify hash
	pow := NewProofOfWork(block, n.config.Settings)

	valid, err := pow.Validate()

	if err != nil {
		return err
	}

	if !valid {
		return errors.New("Block hash is not valid")
	}
	n.Logger.Trace.Println("Block hash verified")
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

	if flags&lib.TXFlagsSkipSQLBaseCheckIfNotOnTop > 0 {
		// check if this block will go to top or no
		lastHash, _, err := n.getBlockchainManager().GetState()

		if err != nil {
			return err
		}

		if bytes.Compare(lastHash, block.PrevBlockHash) != 0 {
			// this is not adding to current top, skip verify of SQL base TXs
			flags = flags | lib.TXFlagsSkipSQLBaseCheck
		}
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
		// we add this frag to ignore error when a row is not in a DB yet. because a row can be inserted in same block before
		// TODO . maybe there is 100% good way to handle this case
		flags = flags | lib.TXFlagsVerifyAllowMissed
		// It can be that a table create is part of this block
		// TODO We have to ensure there is table create query as part of this bloc
		flags = flags | lib.TXFlagsVerifyAllowTableMissed

		err := n.VerifyTransaction(&tx, prevTXs, block.PrevBlockHash, block.Height-1, flags)

		if err != nil {
			n.Logger.Trace.Printf("tx verify error %x  %s", tx.GetID(), err.Error())
			return err
		}

		n.Logger.Trace.Printf("checked %x . add it to previous list", tx.GetID())
		prevTXs = append(prevTXs, tx)
	}
	// 1.
	if !coinbaseused {
		return errors.New("No coinbase TX in the block")
	}
	return nil
}

// Verify transaction against all rules
func (n NodeBlockMaker) VerifyTransaction(tx *structures.Transaction, prevTXs []structures.Transaction,
	prevBlockHash []byte, prevBlockHeight int, flags int) error {

	// check if provided tip is top of chain or no
	isOnTop := false

	var curBlockHash []byte
	var curBlockHeight int
	var err error

	if len(prevBlockHash) > 0 {
		curBlockHash, curBlockHeight, err = n.getBlockchainManager().GetState()

		if err != nil {
			return err
		}

		if bytes.Compare(curBlockHash, prevBlockHash) == 0 {
			isOnTop = true
		}
	} else {
		isOnTop = true
	}

	if isOnTop {
		flags = flags | lib.TXFlagsBasedOnTopOfChain
	}
	n.Logger.Trace.Printf("Go to verify in TXMan %x flags %d", tx.GetID(), flags)
	vtx, err := n.getTransactionsManager().VerifyTransaction(tx, prevTXs, prevBlockHash, flags)

	if err != nil {
		return err
	}

	if !vtx {
		return errors.New(fmt.Sprintf("Transaction in a block is not valid: %x", tx.GetID()))
	}

	if tx.IsSQLCommand() {
		//n.Logger.Trace.Printf("Go to parse %x , flags %d", tx.GetID(), flags)
		qparsed, err := n.parseQueryFromTX(tx, flags)

		if err != nil {
			n.Logger.Trace.Printf("Error TX parsing %s", err.Error())
			return err
		}

		if prevBlockHeight < 0 {
			if len(curBlockHash) > 0 {
				prevBlockHash = curBlockHash
				prevBlockHeight = curBlockHeight
			} else {
				// get current blockheight
				prevBlockHash, prevBlockHeight, err = n.getBlockchainManager().GetState()

				if err != nil {
					return err
				}
			}

		}
		// check execution permissions to ensure this SQL operation is allowed
		err = n.verifyTransactionSQLPermissions(tx, qparsed, prevBlockHeight)

		if err != nil {
			return err
		}
		// check if paid part is correct. contains correct amount anddestination address

		err = n.verifyTransactionPaidSQL(tx, qparsed, prevBlockHeight, flags)

		if err != nil {
			return err
		}
	}

	return nil
}

// Add transaction to a pool. This will call verification and if all is ok it adds to a poool of transactions
func (n *NodeBlockMaker) AddTransactionToPool(tx *structures.Transaction, flags int) error {
	// TODO . put local verification than call adding with TX manager
	// as list of all previous TXs we have current pool state

	if tx.IsCoinbaseTransfer() {
		// we don't add this
		n.Logger.Trace.Println("It is coin base")
		return nil
	}

	err := n.VerifyTransaction(tx, nil, []byte{}, -1, flags)

	if err != nil {
		return err
	}

	return n.getTransactionsManager().AddNewTransaction(tx, flags)
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

	//n.Logger.Trace.Printf("TX count limits %d - %d", min, max)
	return min, max, nil
}
func (n NodeBlockMaker) parseQueryFromTX(tx *structures.Transaction, flags int) (*dbquery.QueryParsed, error) {
	qp := n.getQueryParser()
	// this will get sql type and data from comments. data can be pubkey, txBytes, signature
	qparsed, err := qp.ParseQuery(tx.GetSQLQuery(), flags)

	if err != nil {
		return nil, err
	}

	return &qparsed, nil
}

//Verify SQL paid transaction. This checks if output is locked to correct address and amount is vald for paid SQL
func (n *NodeBlockMaker) verifyTransactionPaidSQL(tx *structures.Transaction, qparsed *dbquery.QueryParsed, prevBlockHeight int, flags int) error {
	// if it is SQL transaction and includes currency part
	// that we must check if a TX was posted to correct destination address
	if !(tx.IsSQLCommand() && tx.IsCurrencyTransfer()) {
		return nil
	}
	paidTXPubKeyHash := n.config.GetPaidTransactionsWalletPubKeyHash()

	if len(paidTXPubKeyHash) == 0 {
		return nil
	}

	// if there is specific address inconsensus rules to send money for SQL updates
	// check also if amount is according to consensus rules
	// possible hashes to send money in this transaction
	possibleHashes := [][]byte{paidTXPubKeyHash}

	byPubKeyHash, err := utils.HashPubKey(tx.ByPubKey)

	if err != nil {
		return errors.New(fmt.Sprintf("TX verify error. Getting TX author pub key hash failed: %s", err.Error()))
	}

	possibleHashes = append(possibleHashes, byPubKeyHash)

	err = structures.CheckTXOutputsAreOnlyToGivenAddresses(tx, possibleHashes)

	if err != nil {
		return err
	}

	if qparsed == nil {
		qparsed, err = n.parseQueryFromTX(tx, flags)
		if err != nil {
			return err
		}

	}

	// check amount

	amount, err := n.getVerifyManager(prevBlockHeight).CheckQueryNeedsPayment(qparsed)

	if err != nil {
		return err
	}

	err = structures.CheckTXOutputValueToAddress(tx, paidTXPubKeyHash, amount)

	if err != nil {
		n.Logger.Trace.Println("Failed output check")
		n.Logger.Trace.Println(tx)
		return err
	}

	return nil
}

//Verify SQL can be executed. This checks if there are permissions to execute this SQL at this point of blockchain
func (n NodeBlockMaker) verifyTransactionSQLPermissions(tx *structures.Transaction, qparsed *dbquery.QueryParsed, prevBlockHeight int) error {
	// if it is SQL transaction and includes currency part
	// that we must check if a TX was posted to correct destination address
	if !tx.IsSQLCommand() {
		return nil
	}

	hasPerm, err := n.getVerifyManager(prevBlockHeight).CheckExecutePermissions(qparsed, tx.ByPubKey)

	if err != nil {
		return err
	}

	if !hasPerm {
		return errors.New("No permissions to execute this query")
	}

	return nil
}
