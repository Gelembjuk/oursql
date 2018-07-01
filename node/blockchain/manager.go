package blockchain

import (
	"bytes"
	"errors"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
)

const (
	BCBAddState_error                = 0
	BCBAddState_addedToTop           = 1
	BCBAddState_addedToParallelTop   = 2
	BCBAddState_addedToParallel      = 3
	BCBAddState_notAddedNoPrev       = 4
	BCBAddState_notAddedExists       = 5
	BCBState_error                   = -1
	BCBState_canAdd                  = 0
	BCBState_exists                  = 1
	BCBState_notExistAndPrevNotExist = 2
)

// Structure to work with blockchain DB

type Blockchain struct {
	DB              database.DBManager
	Logger          *utils.LoggerMan
	HashCache       map[string]int
	LastHashInCache []byte
}

func NewBlockchainManager(DB database.DBManager, Logger *utils.LoggerMan) (*Blockchain, error) {
	bc := Blockchain{}
	bc.DB = DB
	bc.Logger = Logger
	bc.HashCache = make(map[string]int)
	bc.LastHashInCache = []byte{}

	return &bc, nil
}

/*
* Add new block to the blockchain
	BCBAddState_error              = 0 not added to the chain. Because of error
	BCBAddState_addedToTop         = 1 added to the top of current chain
	BCBAddState_addedToParallelTop = 2 added to the top, but on other branch. Other branch becomes primary now
	BCBAddState_addedToParallel    = 3 added but not in main branch and heigh i lower then main branch
	BCBAddState_notAddedNoPrev     = 4 previous not found
	BCBAddState_notAddedExists     = 5 already in blockchain
*
*/
func (bc *Blockchain) AddBlock(block *structures.Block) (uint, error) {
	bc.Logger.Trace.Printf("Adding new block to block chain %x", block.Hash)

	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return BCBAddState_error, err
	}

	blockInDb, err := bcdb.GetBlock(block.Hash)

	if err != nil {
		return BCBAddState_error, err
	}

	if blockInDb != nil {
		return BCBAddState_notAddedExists, nil // already in blockchain
	}

	prevBlockData, err := bcdb.GetBlock(block.PrevBlockHash)

	if err != nil {
		return BCBAddState_error, err
	}

	if prevBlockData == nil {
		// previous block is not yet in our DB
		return BCBAddState_notAddedNoPrev, nil // means block is not added because previous is not in the DB
	}

	// add this block
	blockData, err := block.Serialize()

	if err != nil {
		return BCBAddState_error, err
	}

	err = bcdb.PutBlock(block.Hash, blockData)

	if err != nil {
		return BCBAddState_error, err
	}
	// get current top hash
	lastHash, err := bcdb.GetTopHash()

	if err != nil {
		return BCBAddState_error, err
	}
	// and top block
	lastBlockData, err := bcdb.GetBlock(lastHash)

	if err != nil {
		return BCBAddState_error, err
	}

	lastBlock, err := structures.NewBlockFromBytes(lastBlockData)

	if err != nil {
		return BCBAddState_error, err
	}

	bc.Logger.Trace.Printf("Current BC state %d , %x\n", lastBlock.Height, lastHash)
	bc.Logger.Trace.Printf("New block height %d\n", block.Height)

	if block.Height > lastBlock.Height {
		// the block becomes highest and is top of he blockchain
		err = bcdb.SaveTopHash(block.Hash)

		if err != nil {
			return BCBAddState_error, err

		}
		bc.Logger.Trace.Printf("Set new current hash %x\n", block.Hash)

		if bytes.Compare(lastHash, block.PrevBlockHash) != 0 {
			// update chain records. it keeps only main chain.
			// it should replace branches now
			bc.Logger.Trace.Printf("Replace branches . previous head is %x", lastHash)
			err := bc.UpdateChainOnNewBranch(lastHash)

			if err != nil {
				bc.Logger.Trace.Printf("Chain replace error %s", err.Error())
			}
			// other branch becomes main branch now.
			// it is needed to reindex unspent transactions and non confirmed
			return BCBAddState_addedToParallelTop, nil // added to the top, but on other branch
		} else {
			// update blocks index. this is "normal" case
			bc.Logger.Trace.Printf("Add %x after %x", block.Hash, block.PrevBlockHash)
			err := bcdb.AddToChain(block.Hash, block.PrevBlockHash)

			if err != nil {
				bc.Logger.Trace.Printf("Chain add error %s", err.Error())
			}
			return BCBAddState_addedToTop, nil
		}
	}
	// block added, but is not on the top
	return BCBAddState_addedToParallel, nil
}

//
// NOTE . Operation is done in memory. On practice we don't expect to have million of
// hashes in different branch. it can be 1-5 . Depends on consensus can be more. but not millions
// top hash is already update when we execute this
func (bc *Blockchain) UpdateChainOnNewBranch(prevTopHash []byte) error {
	// go over blocks sttarting from top till bock is found in chain
	newBlocks := []*structures.BlockShort{}

	bc.Logger.Trace.Printf("UCONB start %x", prevTopHash)

	bci, err := NewBlockchainIterator(bc.DB)

	if err != nil {
		return err
	}

	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return err
	}

	// this is the hash that exists in chain. intersaction of old and new branch
	mergePointHash := []byte{}

	for {
		block, _ := bci.Next()

		//bc.Logger.Trace.Printf("UCONB load from new %x", block.Hash)

		exists, err := bcdb.BlockInChain(block.Hash)

		if err != nil {
			return err
		}

		if exists {
			mergePointHash = block.Hash[:]
			//bc.Logger.Trace.Printf("UCONB it exists %x", mergePointHash)
			break
		}

		newBlocks = append(newBlocks, block.GetShortCopy())

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	remHash := prevTopHash
	// remove back old bbranch from chain
	for {
		//bc.Logger.Trace.Printf("UCONB go to remove %x", remHash)
		if bytes.Compare(remHash, mergePointHash) == 0 {
			// top removing
			bc.Logger.Trace.Printf("UCONB time to exit")
			break
		}
		_, prevHash, _, err := bcdb.GetLocationInChain(remHash)

		if err != nil {
			return err
		}

		err = bcdb.RemoveFromChain(remHash)
		//bc.Logger.Trace.Printf("UCONB removed %x", remHash)
		if err != nil {
			return err
		}

		remHash = prevHash[:]
	}
	// at this point , mergePointHash is top in the chain records

	structures.ReverseBlocksShortSlice(newBlocks)

	for _, block := range newBlocks {
		//bc.Logger.Trace.Printf("UCONB add %x after %x", block.Hash, mergePointHash)
		err := bcdb.AddToChain(block.Hash, mergePointHash)

		if err != nil {
			return err
		}

		mergePointHash = block.Hash[:]
	}

	return nil
}

/*
* DeleteBlock deletes the top block (last added)
* The function extracts the last block, deletes it and sets the tip to point to
* previous block.
* TODO
* It is needed to make some more correct logic. f a block is removed then tip could go to some other blocks branch that
* is longer now. It is needed to care blockchain branches
* Returns deleted block object
 */
func (bc *Blockchain) DeleteBlock() (*structures.Block, error) {
	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return nil, err
	}

	blockInDb, err := bcdb.GetTopBlock()

	if err != nil {
		return nil, err
	}

	if blockInDb == nil {
		return nil, errors.New("Top block is not found!")
	}

	block, err := structures.NewBlockFromBytes(blockInDb)

	if err != nil {
		return nil, err
	}

	err = bcdb.SaveTopHash(block.PrevBlockHash)

	if err != nil {
		return nil, err
	}

	err = bcdb.DeleteBlock(block.Hash)

	if err != nil {
		return nil, err
	}

	bcdb.RemoveFromChain(block.Hash)

	return block, nil
}

// GetTransactionFromBlock finds a transaction by its ID in given block
// If block is known . It worsk much faster then FindTransaction
func (bc *Blockchain) GetTransactionFromBlock(txID []byte, blockHash []byte) (structures.TransactionInterface, error) {
	block, err := bc.GetBlock(blockHash)

	if err != nil {
		return nil, err
	}

	// get transaction from a block
	for _, tx := range block.Transactions {
		if bytes.Compare(tx.GetID(), txID) == 0 {
			return tx, nil
		}
	}

	return nil, errors.New("Transaction is not found")
}

// Returns a block with specified height in current blockchain
// TODO can be optimized using blocks index
func (bc *Blockchain) GetBlockAtHeight(height int) (*structures.Block, error) {
	// finds a block with this height

	bci, err := NewBlockchainIterator(bc.DB)

	if err != nil {
		return nil, err
	}

	for {
		block, _ := bci.Next()

		if block.Height == height {
			return block, nil
		}

		if block.Height < height {
			return nil, errors.New("Block with the heigh doesn't exist")
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return nil, errors.New("Block with the heigh doesn't exist")
}

// GetBestHeight returns the height of the latest block

func (bc *Blockchain) GetBestHeight() (int, error) {
	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return 0, err
	}

	blockData, err := bcdb.GetTopBlock()

	if err != nil {
		return 0, err
	}

	lastBlock, err := structures.NewBlockFromBytes(blockData)

	if err != nil {
		return 0, err
	}

	return lastBlock.Height, nil
}

// Returns info about the top block. Hash and Height

func (bc *Blockchain) GetState() ([]byte, int, error) {
	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return nil, 0, err
	}

	blockData, err := bcdb.GetTopBlock()

	if err != nil {
		return nil, 0, err
	}

	lastBlock, err := structures.NewBlockFromBytes(blockData)

	if err != nil {
		return nil, 0, err
	}

	return lastBlock.Hash, lastBlock.Height, nil
}

// Check block exists

func (bc *Blockchain) CheckBlockExists(blockHash []byte) (bool, error) {
	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return false, err
	}

	return bcdb.CheckBlockExists(blockHash)
}

// GetBlock finds a block by its hash and returns i
func (bc *Blockchain) GetBlock(blockHash []byte) (structures.Block, error) {
	var block structures.Block

	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return block, err
	}

	blockData, err := bcdb.GetBlock(blockHash)

	if err != nil {
		return block, err
	}

	if blockData == nil {
		return block, errors.New("Block is not found.")
	}
	blocktmp, err := structures.NewBlockFromBytes(blockData)

	if err != nil {
		return block, err
	}

	block = *blocktmp.Copy()

	return block, nil
}

// Returns a list of blocks short info stating from given block or from a top
func (bc *Blockchain) GetBlocksShortInfo(startfrom []byte, maxcount int) []*structures.BlockShort {
	var blocks []*structures.BlockShort
	var bci *BlockchainIterator

	var err error

	if len(startfrom) > 0 {
		bci, err = NewBlockchainIteratorFrom(bc.DB, startfrom)

	} else {
		bci, err = NewBlockchainIterator(bc.DB)
	}

	if err != nil {
		return blocks
	}

	for {
		block, _ := bci.Next()
		bs := block.GetShortCopy()

		blocks = append(blocks, bs)

		if len(block.PrevBlockHash) == 0 {
			break
		}

		if len(blocks) > maxcount {
			break
		}
	}

	return blocks
}

// returns a list of hashes of all the blocks in the chain
func (bc *Blockchain) GetNextBlocks(startfrom []byte) ([]*structures.BlockShort, error) {
	localError := func(err error) ([]*structures.BlockShort, error) {
		return nil, err
	}

	maxcount := 1000

	blocks := []*structures.BlockShort{}

	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return localError(err)
	}

	hash := startfrom[:]

	for {
		_, _, nextHash, err := bcdb.GetLocationInChain(hash)

		if err != nil {
			return localError(err)
		}

		block, err := bc.GetBlock(hash)

		if err != nil {
			return localError(err)
		}

		blocks = append(blocks, block.GetShortCopy())

		if len(blocks) >= maxcount {
			break
		}
		if len(nextHash) == 0 {
			break
		}
		hash = nextHash[:]
	}

	if len(blocks) == 0 {
		return nil, nil
	}

	return blocks, nil
}

// Returns first blocks in block chain
func (bc *Blockchain) GetFirstBlocks(maxcount int) ([]*structures.Block, int, error) {
	localError := func(err error) ([]*structures.Block, int, error) {
		return nil, 0, err
	}
	_, height, err := bc.GetState()

	if err != nil {
		return localError(err)
	}

	genesisHash, err := bc.GetGenesisBlockHash()

	if err != nil {
		bc.Logger.Trace.Println("err 2")
		return localError(err)
	}

	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return localError(err)
	}

	blocks := []*structures.Block{}

	hash := genesisHash[:]

	for {
		_, _, nextHash, err := bcdb.GetLocationInChain(hash)

		if err != nil {
			return localError(err)
		}

		block, err := bc.GetBlock(hash)

		if err != nil {
			return localError(err)
		}

		blocks = append(blocks, &block)

		if len(blocks) >= maxcount {
			break
		}
		if len(nextHash) == 0 {
			break
		}
		hash = nextHash[:]
	}
	return blocks, height, nil
}

// Returns a chain of blocks starting from a hash and till
// end of blockchain or block from main chain found
// if already in main chain then returns empty list
// Returns also a block from main chain which is the base of the side branch
//
// The function load all hashes to the memory from "main" chain

func (bc *Blockchain) GetSideBranch(hash []byte, currentTip []byte) ([]*structures.Block, []*structures.Block, *structures.Block, error) {
	localError := func(err error) ([]*structures.Block, []*structures.Block, *structures.Block, error) {
		return nil, nil, nil, err
	}
	// get 2 blocks with hashes from arguments
	sideblock_o, err := bc.GetBlock(hash)

	if err != nil {
		return localError(err)
	}

	topblock_o, err := bc.GetBlock(currentTip)

	if err != nil {
		return localError(err)
	}

	sideblock := &sideblock_o
	topblock := &topblock_o

	bc.Logger.Trace.Printf("States: top %d, side %d", topblock.Height, sideblock.Height)

	if sideblock.Height < 1 || topblock.Height < 1 {
		return localError(errors.New("Can not do this for genesis block"))
	}

	sideBlocks := []*structures.Block{}
	mainBlocks := []*structures.Block{}

	if sideblock.Height > topblock.Height {
		// go down from side block till heigh is same as top
		bci, err := NewBlockchainIteratorFrom(bc.DB, sideblock.Hash)

		if err != nil {
			return localError(err)
		}

		for {
			block, _ := bci.Next()
			bc.Logger.Trace.Printf("next side %x", block.Hash)
			if block.Height == topblock.Height {
				sideblock = block
				break
			}
			sideBlocks = append(sideBlocks, block)
		}
	} else if sideblock.Height < topblock.Height {
		// go down from top block till heigh is same as side
		bci, err := NewBlockchainIteratorFrom(bc.DB, topblock.Hash)

		if err != nil {
			return localError(err)
		}

		for {
			block, _ := bci.Next()
			bc.Logger.Trace.Printf("next top %x", block.Hash)
			if block.Height == sideblock.Height {
				topblock = block
				break
			}
			mainBlocks = append(mainBlocks, block)
		}
	}

	// at this point sideblock and topblock have same heigh
	bcis, err := NewBlockchainIteratorFrom(bc.DB, sideblock.Hash)

	if err != nil {
		return localError(err)
	}
	bcit, err := NewBlockchainIteratorFrom(bc.DB, topblock.Hash)

	if err != nil {
		return localError(err)
	}

	for {
		sideblock, _ = bcis.Next()
		topblock, _ = bcit.Next()

		bc.Logger.Trace.Printf("parallel %x vs %x", sideblock.Hash, topblock.Hash)

		if bytes.Compare(sideblock.Hash, topblock.Hash) == 0 {

			structures.ReverseBlocksSlice(mainBlocks)

			return sideBlocks, mainBlocks, sideblock, nil
		}
		// side blocks are returned in same order asthey are
		// main blocks must be reversed to add them in correct order
		mainBlocks = append(mainBlocks, topblock)
		sideBlocks = append(sideBlocks, sideblock)

		if len(sideblock.PrevBlockHash) == 0 || len(topblock.PrevBlockHash) == 0 {
			return localError(errors.New("No connect with main blockchain"))
		}

	}
	// this point should be never reached
	return nil, nil, nil, errors.New("Chain error")
}

/*
* Returns a chain of blocks starting from a hash and till
* end of blockchain or block from main chain found
* if already in main chain then returns empty list
*
* The function load all hashes to the memory from "main" chain
* TODO We need to use index of blocks
 */
func (bc *Blockchain) GetBranchesReplacement(sideBranchHash []byte, tip []byte) ([]*structures.Block, []*structures.Block, error) {
	bc.Logger.Trace.Printf("Go to get branch %x %x", sideBranchHash, tip)

	sideBlocks, mainBlocks, BCBlock, err := bc.GetSideBranch(sideBranchHash, tip)

	bc.Logger.Trace.Printf("Result sideblocks %d mainblocks %d", len(sideBlocks), len(mainBlocks))
	bc.Logger.Trace.Printf("%x", BCBlock.Hash)

	if err != nil {
		return nil, nil, err
	}

	if bytes.Compare(BCBlock.Hash, sideBranchHash) == 0 {
		// side branch is part of the tip chain
		return nil, nil, nil
	}
	bc.Logger.Trace.Println("Main blocks")
	for _, b := range mainBlocks {
		bc.Logger.Trace.Printf("%x", b.Hash)
	}
	bc.Logger.Trace.Println("Side blocks")
	for _, b := range sideBlocks {
		bc.Logger.Trace.Printf("%x", b.Hash)
	}
	return mainBlocks, sideBlocks, nil
}

// Returns genesis block hash
// First block is saved in sperate record in DB
func (bc *Blockchain) GetGenesisBlockHash() ([]byte, error) {
	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return nil, err
	}

	genesisHash, err := bcdb.GetFirstHash()

	if err != nil {
		return nil, err
	}

	return genesisHash, nil
}

// Receive ist of hashes and return a hash that is in the chain defined by top hash
// This function is used for transactions. to understand if a transaction exists in current chain branch or in some parallel
// and where a transaction is spent
func (bc *Blockchain) ChooseHashUnderTip(blockHashes [][]byte, topHash []byte) ([]byte, error) {
	//bc.Logger.Trace.Printf("choose block hash %x for top  %x", blockHashes, topHash)
	if len(blockHashes) == 0 {
		return nil, nil
	}

	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return nil, err
	}

	if len(topHash) == 0 {
		// try to find under primary branch

		for _, blockHash := range blockHashes {
			exists, err := bcdb.BlockInChain(blockHash)

			if err != nil {
				return nil, err
			}
			if exists {
				// this block hash is under main branch. return it
				return blockHash, nil
			}
		}
		// noone of blocks is under main hash
		return nil, nil
	}
	// ,aybe one of hashes is equal to top
	for _, blockHash := range blockHashes {
		if bytes.Compare(blockHash, topHash) == 0 {
			return blockHash, nil
		}
	}
	// get top block. we will need info about it below
	topBlock, err := bc.GetBlock(topHash)

	if err != nil {
		return nil, err
	}

	// check if top hash is in the main chain
	exists, err := bcdb.BlockInChain(topHash)

	if err != nil {
		return nil, err
	}

	if !exists {
		// our top block is not part of primary chain
		// we go down in blocks while find one of blocks
		// or while find first block in the chain. when we find in the chain
		// we use it as new top

		bci, err := NewBlockchainIteratorFrom(bc.DB, topHash)

		if err != nil {
			return nil, err
		}

		foundnewtop := false

		for {
			block, err := bci.Next()

			if err != nil {
				return nil, err
			}

			// check if this block is equal to any requested
			for _, blockHash := range blockHashes {
				if bytes.Compare(blockHash, block.Hash) == 0 {
					// it is found
					return blockHash, nil
				}
			}

			// check if this block is part of main branch
			exists, err := bcdb.BlockInChain(block.Hash)

			if err != nil {
				return nil, err
			}
			if exists {
				// we now can consider this block as new top and continue with it in next section
				topBlock = *block

				foundnewtop = true
				break
			}

			// check if this is the end of blockchain
			if len(block.PrevBlockHash) == 0 {
				break
			}
		}

		if !foundnewtop {
			return nil, nil
		}
	}

	// top block is part of  main chain
	// find in primary chain but it must be lower height than current
	for _, blockHash := range blockHashes {
		exists, err := bcdb.BlockInChain(blockHash)

		if err != nil {
			return nil, err
		}
		if exists {
			// this block hash is under main branch too
			// check if height is lower then top
			block, err := bc.GetBlock(blockHash)

			if err != nil {
				return nil, err
			}

			if block.Height < topBlock.Height {
				// this block is under primary chain too and is lower then top
				return blockHash, nil
			}

		}
	}
	// no block found
	return nil, nil
}

// Receive ist of hashes and return a hash that is in the chain defined by top hash
func (bc *Blockchain) CheckBlockIsInRange(blockHash []byte, bottomHash []byte, topHash []byte) (bool, error) {

	bcdb, err := bc.DB.GetBlockchainObject()

	if err != nil {
		return false, err
	}

	if len(topHash) > 0 {
		// this can be block out of main branch

		exists, err := bcdb.BlockInChain(topHash)

		if err != nil {
			return false, err
		}

		if !exists {
			// block is not in main chain
			bci, err := NewBlockchainIteratorFrom(bc.DB, topHash)

			if err != nil {
				return false, err
			}

			for {
				block, err := bci.Next()

				if err != nil {
					return false, err
				}

				if bytes.Compare(blockHash, block.Hash) == 0 {
					// it is found
					return true, nil
				}

				if bytes.Compare(bottomHash, block.Hash) == 0 {
					// not found
					return false, nil
				}

				// check if this block is part of main branch
				exists, err := bcdb.BlockInChain(block.Hash)

				if err != nil {
					return false, err
				}
				if exists {
					// we now can consider this block as new top and continue with it in next section
					topHash = block.Hash[:]
					break
				}

				// check if this is the end of blockchain
				if len(block.PrevBlockHash) == 0 {
					// we browsed al chain and no bottom found
					return false, nil
				}
			}
		}
	} else {
		// load top hash
		topHash, err = bcdb.GetTopHash()

		if err != nil {
			return false, err
		}
		bc.Logger.Trace.Printf("found top %x", topHash)
	}
	// ensure our block is also in main branch
	exists, err := bcdb.BlockInChain(blockHash)

	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}
	// at this point we know top and bottom hash are both in main branch. we only load all 3 blocks
	// to get height and compare

	// get top block.
	topBlock, err := bc.GetBlock(topHash)

	if err != nil {
		return false, err
	}

	bottomBlock, err := bc.GetBlock(bottomHash)

	if err != nil {
		return false, err
	}

	block, err := bc.GetBlock(blockHash)

	if err != nil {
		return false, err
	}

	if block.Height >= bottomBlock.Height && block.Height <= topBlock.Height {
		return true, nil
	}

	return false, nil
}
