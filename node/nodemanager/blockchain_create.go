package nodemanager

import (
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
	DBConn          *Database
	consensusConfig *consensus.ConsensusConfig
}

// Blockchain DB manager object
func (n *makeBlockchain) getBCManager() *blockchain.Blockchain {
	bcm, _ := blockchain.NewBlockchainManager(n.DBConn.DB(), n.Logger)
	return bcm
}

// Transactions manager object
func (n *makeBlockchain) getTransactionsManager() transactions.TransactionsManagerInterface {
	return transactions.NewManager(n.DBConn.DB(), n.Logger, n.consensusConfig.GetInfoForTransactions())
}

// Init block maker object. It is used to make new blocks
func (n *makeBlockchain) getBlockMakeManager() (consensus.BlockMakerInterface, error) {
	return consensus.NewBlockMakerManager(n.consensusConfig, n.MinterAddress, n.DBConn.DB(), n.Logger)
}

// Create new blockchain, add genesis block witha given text
func (n *makeBlockchain) CreateBlockchain() error {
	// firstly, check if DB is accesible
	n.Logger.Trace.Printf("Check DB connection is fine")
	err := n.DBConn.CheckConnection()
	if err != nil {
		return err
	}

	genesisCoinbaseData := "some string. this is TEMP"
	genesisBlock, err := n.prepareGenesisBlock(n.MinterAddress, genesisCoinbaseData)

	if err != nil {
		return err
	}

	Minter, _ := n.getBlockMakeManager()

	n.Logger.Trace.Printf("Complete genesis block proof of work\n")

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

	n.Logger.Trace.Printf("Blockchain ready!\n")

	return nil
}

// Creates new blockchain DB from given list of blocks
// This would be used when new empty node started and syncs with other nodes

func (n *makeBlockchain) InitBlockchainFromOther(addr net.NodeAddr, nodeclient *nodeclient.NodeClient, BC *NodeBlockchain) (bool, error) {
	n.Logger.Trace.Printf("Check DB connection is fine")
	err := n.DBConn.CheckConnection()

	if err != nil {
		// no sence to continue
		return false, err
	}
	n.Logger.Trace.Printf("Try to init blockchain from %s:%d", addr.Host, addr.Port)

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

			_, err = BC.AddBlock(block)

			if err != nil {
				return false, err
			}

			TXMan.BlockAdded(block, true)

			MH = block.Height
		}
	}

	return MH == result.Height, nil
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

	n.Logger.Trace.Printf("Prepare TX caches\n")

	n.getTransactionsManager().BlockAdded(genesis, true)

	return err
}
