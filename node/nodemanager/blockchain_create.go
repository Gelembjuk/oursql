package nodemanager

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/lib/nodeclient"
	"github.com/gelembjuk/oursql/lib/remoteclient"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/blockchain"
	"github.com/gelembjuk/oursql/node/consensus"
	"github.com/gelembjuk/oursql/node/structures"
	"github.com/gelembjuk/oursql/node/transactions"
)

type makeBlockchain struct {
	Logger          *utils.LoggerMan
	MinterAddress   string
	PubKey          []byte
	PrivateKey      ecdsa.PrivateKey
	BC              *NodeBlockchain
	DBConn          *Database
	consensusConfig *consensus.ConsensusConfig
}

// Blockchain DB manager object
func (n *makeBlockchain) getBCManager() *blockchain.Blockchain {
	bcm, _ := blockchain.NewBlockchainManager(n.DBConn.DB(), n.Logger)
	return bcm
}

// Init SQL transactions manager
func (n *makeBlockchain) getSQLQueryManager() (consensus.SQLTransactionsInterface, error) {
	return consensus.NewSQLQueryManager(n.consensusConfig, n.DBConn.DB(), n.Logger, n.PubKey, n.PrivateKey)
}

// Transactions manager object
func (n *makeBlockchain) getTransactionsManager() transactions.TransactionsManagerInterface {
	return transactions.NewManager(n.DBConn.DB(), n.Logger, n.consensusConfig.GetInfoForTransactions())
}

// Init block maker object. It is used to make new blocks
func (n *makeBlockchain) getBlockMakeManager() consensus.BlockMakerInterface {
	return consensus.NewBlockMakerManager(n.consensusConfig, n.MinterAddress, n.DBConn.DB(), n.Logger)
}

// Create new blockchain, add genesis block witha given text
func (n *makeBlockchain) CreateBlockchain(genesisCoinbaseData string, initOnNotEmpty bool) error {
	// firstly, check if DB is accesible
	n.Logger.Trace.Printf("Check DB connection is fine")
	tables, err := n.DBConn.GetAllTables()

	if err != nil {
		return err
	}
	tables = n.filterTablesForManaged(tables)

	// check if a DB is empty or not. load list of existent tables
	if len(tables) > 0 && !initOnNotEmpty {
		return errors.New("The DB is not empty. It is not allowed to init on non-empty DB")
	}

	genesisBlock, err := n.prepareGenesisBlock(n.MinterAddress, genesisCoinbaseData)

	if err != nil {
		return err
	}

	Minter := n.getBlockMakeManager()

	//n.Logger.Trace.Printf("Complete genesis block proof of work\n")

	Minter.SetPreparedBlock(genesisBlock)

	genesisBlock, err = Minter.CompleteBlock()

	if err != nil {
		return err
	}

	n.Logger.Trace.Printf("Block ready. Init block chain file\n")

	err = n.addFirstBlock(genesisBlock)

	if err != nil {
		return err
	}

	n.Logger.Trace.Printf("Blockchain ready! Now looks existent tables\n")

	if len(tables) > 0 {
		countTables, countRows, err := n.addExistentTables(tables)

		if err != nil {
			return err
		}

		n.Logger.Trace.Printf("Existent tables added fine. %d tables and %d total rows", countTables, countRows)
	}

	return nil
}

// Creates new blockchain DB from given list of blocks
// This would be used when new empty node started and syncs with other nodes

func (n *makeBlockchain) InitBlockchainFromOther(addr net.NodeAddr, nodeclient *nodeclient.NodeClient) (bool, error) {
	n.Logger.Trace.Printf("Check DB connection is fine")
	err := n.DBConn.CheckConnection()

	if err != nil {
		// no sence to continue
		return false, err
	}
	n.Logger.Trace.Printf("Try to init blockchain from %s:%d", addr.Host, addr.Port)

	err = n.importBlockchainConsensusInfo(addr, nodeclient)

	if err != nil {
		return false, err
	}

	result, err := nodeclient.SendGetFirstBlocks(addr)

	if err != nil {
		return false, err
	}

	if len(result.Blocks) == 0 {
		return false, errors.New("No blocks found on taht node")
	}

	firstblockbytes := result.Blocks[0]

	block, err := structures.NewBlockFromBytes(firstblockbytes)

	if err != nil {
		return false, err
	}

	n.Logger.Trace.Printf("Importing first block hash %x", block.Hash)
	// make blockchain with single block
	err = n.addFirstBlock(block)

	if err != nil {
		return false, errors.New(fmt.Sprintf("Create DB abd add first block: %", err.Error()))
	}

	defer n.DBConn.CloseConnection()

	MH := block.Height

	TXMan := n.getTransactionsManager()

	if len(result.Blocks) > 1 {
		// add all blocks

		skip := true
		for _, blockdata := range result.Blocks {
			if skip {
				skip = false
				continue
			}
			// add this block
			block, err := structures.NewBlockFromBytes(blockdata)

			if err != nil {
				return false, err
			}

			_, err = n.BC.AddBlock(block)

			if err != nil {
				return false, err
			}

			TXMan.BlockAdded(block, true)

			MH = block.Height
		}
	}

	return MH == result.Height, nil
}

// Import blockchain consensus info. Config file, plugin etc
func (n *makeBlockchain) importBlockchainConsensusInfo(fromnode net.NodeAddr, nodeclient *nodeclient.NodeClient) error {
	// load consensus info
	result, err := nodeclient.SendGetConsensusData(fromnode)

	if err != nil {
		n.Logger.Error.Printf("Failed to import consensus data %s", err.Error())
		return err
	}
	//n.Logger.Trace.Printf("Loaded consensus file with len %d and contents %s", len(result.ConfigFile), string(result.ConfigFile))
	return n.consensusConfig.UpdateConfig(result.ConfigFile)
}

// BUilds a genesis block. It is used only to start new blockchain
func (n *makeBlockchain) prepareGenesisBlock(address, genesisCoinbaseData string) (*structures.Block, error) {
	if address == "" {
		return nil, errors.New("Geneisis block wallet address missed")
	}

	w := remoteclient.Wallet{}

	if !w.ValidateAddress(address) {
		return nil, errors.New("Address is not valid")
	}

	if genesisCoinbaseData == "" {
		return nil, errors.New("Geneisis block text missed")
	}

	cbtx, errc := structures.NewCoinbaseTransaction(address, genesisCoinbaseData, n.consensusConfig.CoinsForBlockMade)

	if errc != nil {
		return nil, errors.New(fmt.Sprintf("Error creating coinbase TX %s", errc.Error()))
	}

	genesis := &structures.Block{}
	genesis.PrepareNewBlock([]structures.Transaction{*cbtx}, []byte{}, 0)

	return genesis, nil
}

// Create new blockchain from given genesis block
func (n *makeBlockchain) addFirstBlock(genesis *structures.Block) error {
	n.Logger.Trace.Println("Init DB")

	n.DBConn.CloseConnection() // close in case if it was opened before

	err := n.DBConn.InitDatabase()

	if err != nil {
		n.Logger.Error.Printf("Can not init DB: %s", err.Error())
		return err
	}

	bcdb, err := n.DBConn.DB().GetBlockchainObject()

	if err != nil {
		n.Logger.Error.Printf("Can not create conn object: %s", err.Error())
		return err
	}

	blockdata, err := genesis.Serialize()

	if err != nil {
		return err
	}

	err = bcdb.PutBlockOnTop(genesis.Hash, blockdata)

	if err != nil {
		return err
	}

	err = bcdb.SaveFirstHash(genesis.Hash)

	if err != nil {
		return err
	}

	// add first rec to chain list
	err = bcdb.AddToChain(genesis.Hash, []byte{})

	if err != nil {
		return err
	}

	n.getTransactionsManager().BlockAdded(genesis, true)

	return err
}

// return only tables that should be managed with blockchain. skip tables ignored in consensus config
func (n *makeBlockchain) filterTablesForManaged(tables []string) []string {
	managedTables := []string{}

	for _, table := range tables {
		managed := true
		for _, t := range n.consensusConfig.UnmanagedTables {
			if table == t {
				managed = false
				break
			}
		}

		if !managed {
			// table can be inspecial list to add to BC on initial BC create
			// it will be ignored only later but must be present in current state on all nodes
			for _, t := range n.consensusConfig.UnmanagedTablesImport {
				if table == t {
					managed = true
					break
				}
			}
		}

		if managed {
			managedTables = append(managedTables, table)
		}
	}

	return managedTables
}

// Add existent tables to blockchain
func (n *makeBlockchain) addExistentTables(tables []string) (countTales int, countRows int, err error) {
	sqlslist := []string{}

	counts := map[string]int{}
	// we need to know total count to keep minimal values of transactions per block
	totalcount := 0
	totalloaded := 0
	// get counts of rows per table
	for _, table := range tables {
		counts[table], err = n.DBConn.DB().QM().ExecuteSQLCountInTable(table)

		if err != nil {
			return
		}
		totalcount = totalcount + 1 // table create SQL
		totalcount = totalcount + counts[table]
	}

	limit := 1000

	nextBlockHeigh := 1

	var sqls []string

	for _, table := range tables {
		offset := 0

		for offset <= counts[table] {

			limitL := limit - len(sqlslist)

			if limitL < 1 {
				limitL = limit
			}

			sqls, err = n.DBConn.DB().QM().ExecuteSQLTableDump(table, limitL, offset)

			if err != nil {
				return
			}

			sqlslist = append(sqlslist, sqls...)
			totalloaded = totalloaded + len(sqls)

			n.Logger.Trace.Printf("SQLs in list %d, limit %d, totalcount %d, total loaded %d", len(sqlslist), limitL, totalcount, totalloaded)

			offset = offset + len(sqls)
			// check if it is time to make new block
			if len(sqlslist) >= limit || totalcount == totalloaded {
				n.Logger.Trace.Printf("check 1 passed")
				// if not more minimum than nothing to do
				// we should not do new block if remaining part of rows is less than required for next block after current
				if totalcount-totalloaded > nextBlockHeigh+1 || totalcount == totalloaded {
					n.Logger.Trace.Printf("check 2 passed")
					// there are more records to load and it will be anough for next block
					err = n.makeNewInitialBlock(sqlslist, nextBlockHeigh)

					if err != nil {
						return
					}

					sqlslist = []string{}
					nextBlockHeigh = nextBlockHeigh + 1
				}
			}
		}

	}

	if len(sqlslist) > 0 {
		n.Logger.Trace.Printf("Do final block")
		// make final block
		err = n.makeNewInitialBlock(sqlslist, nextBlockHeigh)

		if err != nil {
			return
		}
	}

	return len(tables), totalcount, nil
}

//Make new initial block from prepared SQLs
func (n *makeBlockchain) makeNewInitialBlock(sqls []string, nextBlockHeigh int) error {
	n.Logger.Trace.Printf("DO next block with %d transactions in t")
	n.consensusConfig.ExtendRulesApplyStartHeigh(nextBlockHeigh)

	qm, err := n.getSQLQueryManager()
	if err != nil {
		return err
	}
	for _, sql := range sqls {
		_, err := qm.NewQueryByNodeInit(sql, n.PubKey, n.PrivateKey)

		if err != nil {
			return err
		}
	}

	// make new block
	Minter := consensus.NewBlockMakerManager(n.consensusConfig, n.MinterAddress, n.DBConn.DB(), n.Logger)

	prepres, err := Minter.PrepareNewBlock()

	if err != nil {
		return err
	}

	// close it while doing the proof of work
	n.DBConn.CloseConnection()
	// and close it again in the end of function
	defer n.DBConn.CloseConnection()

	if prepres != consensus.BlockPrepare_Done {
		return errors.New("Block an not be done. Somethign went wrong")
	}

	block, err := Minter.CompleteBlock()

	if err != nil {
		n.Logger.Trace.Printf("Block completion error. %s", err)
		return err
	}

	// add new block to local blockchain. this will check a block again

	_, err = n.BC.AddBlock(block)

	if err != nil {
		n.Logger.Trace.Printf("Block add error. %s", err)
		return err
	}

	n.getTransactionsManager().BlockAdded(block, true)

	return nil
}
