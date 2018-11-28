package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/lib/nodeclient"
	"github.com/gelembjuk/oursql/lib/remoteclient"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/config"
	"github.com/gelembjuk/oursql/node/consensus"
	"github.com/gelembjuk/oursql/node/nodemanager"
	"github.com/gelembjuk/oursql/node/server"
)

var allowWithoutBCReady = []string{"initblockchain",
	"importblockchain",
	"interactiveautocreate",
	"restoreblockchain",
	"createwallet",
	"listaddresses",
	"nodestate"}

var disableWithBCReady = []string{"initblockchain",
	"initblockchain",
	"importblockchain",
	"importandstart",
	"restoreblockchain"}

var commandsInteractiveMode = []string{
	"initblockchain",
	"importblockchain",
	"restoreblockchain",
	"dumpblockchain",
	"exportconsensusconfig",
	"pullupdates",
	"printchain",
	"makeblock",
	"reindexcache",
	"send",
	"sql",
	"getbalance",
	"getbalances",
	"createwallet",
	"listaddresses",
	"unapprovedtransactions",
	"mineblock",
	"canceltransaction",
	"dropblock",
	"addrhistory",
	"showunspent",
	"shownodes",
	"addnode",
	"removenode"}

var commandNodeManageMode = []string{
	"interactiveautocreate",
	"importandstart",
	"startnode",
	"startintnode",
	"stopnode",
	config.Daemonprocesscommandline,
	"nodestate"}

type NodeCLI struct {
	Input                      config.AppInput
	Logger                     *utils.LoggerMan
	ConfigDir                  string
	ConseususConfigFile        string
	ConseususConfigFilePresent bool
	Command                    string
	AlreadyRunningPort         int
	NodeAuthStr                string
	Node                       *nodemanager.Node
}

/*
* Creates a client object
 */
func getNodeCLI(input config.AppInput) NodeCLI {
	cli := NodeCLI{}
	cli.Input = input
	cli.ConfigDir = input.ConfigDir
	cli.Command = input.Command

	cli.Logger = utils.CreateLogger()

	cli.Logger.EnableLogs(input.Logs)

	if input.Args.LogDest != "stdout" {
		cli.Logger.LogToFiles(cli.ConfigDir, "log_trace.txt", "log_traceext.txt", "log_info.txt", "log_warning.txt", "log_error.txt")
	} else {
		cli.Logger.LogToStdout()
	}

	cli.ConseususConfigFile = input.ConseususConfigFile
	cli.ConseususConfigFilePresent = input.ConseususConfigFilePresent

	cli.Node = nil
	// check if Daemon is already running
	nd := server.NodeDaemon{}
	nd.ConfigDir = cli.ConfigDir
	nd.Logger = cli.Logger

	port, authstr := nd.GetRunningProcessInfo()

	cli.AlreadyRunningPort = port
	cli.NodeAuthStr = authstr

	cli.Logger.Trace.Println("Node CLI inited")

	return cli
}

/*
* Createes node object. Node does all work related to acces to bockchain and DB
 */
func (c *NodeCLI) CreateNode() error {
	if c.Node != nil {
		//already created
		return nil
	}
	node := nodemanager.Node{}

	node.ConfigDir = c.ConfigDir

	node.DBConn = &nodemanager.Database{}

	node.DBConn.SetLogger(c.Logger)

	node.DBConn.SetConfig(c.Input.Database)
	// at this place DB doesn't do a connection attempt
	node.DBConn.Init()

	node.Logger = c.Logger
	node.MinterAddress = c.Input.MinterAddress

	var err error
	// load consensus config
	if c.ConseususConfigFilePresent {
		node.ConsensusConfig, err = consensus.NewConfigFromFile(c.ConseususConfigFile)
	} else {
		node.ConsensusConfig, err = consensus.NewConfigDefault()
	}

	if err != nil {
		c.Logger.Error.Printf("Error when init consensus config %s", err.Error())
		return err
	}

	node.ConsensusConfig.SetConfigFilePath(c.Input.ConseususConfigFile)

	node.Init()
	node.InitNodes(c.Input.Nodes, false)

	node.NodeClient.SetAuthStr(c.NodeAuthStr)

	c.Node = &node

	c.setNodeProxyKeys()

	return nil
}

func (c *NodeCLI) getApplicationName() string {
	c.CreateNode()
	return c.Node.ConsensusConfig.Application.Name
}

// Check if there is internal keys pair to sign DB proxy transactions. Attach if it is set
func (c *NodeCLI) setNodeProxyKeys() error {
	c.Node.ProxyPubKey = []byte{}

	if c.Input.ProxyKey != "" {
		walletscli, err := c.getWalletsCLI()

		if err == nil {
			walletobj, err := walletscli.WalletsObj.GetWallet(c.Input.ProxyKey)

			if err == nil {
				c.Node.ProxyPubKey = walletobj.GetPublicKey()
				c.Node.ProxyPrivateKey = walletobj.GetPrivateKey()
			}
		}

	}

	return nil
}

// Detects if this request is not related to node server management and must return response right now
func (c NodeCLI) isInteractiveMode() bool {

	return utils.StringInSlice(c.Command, commandsInteractiveMode)
}

// Detects if it is a node management command
func (c NodeCLI) isNodeManageMode() bool {

	if utils.StringInSlice(c.Command, commandNodeManageMode) {
		return true
	}
	return false
}

// Executes the client command in interactive mode
func (c NodeCLI) ExecuteCommand() error {
	err := c.CreateNode() // init node struct

	if err != nil {
		return err
	}

	bcexists := c.Node.BlockchainExist()

	if !utils.StringInSlice(c.Command, allowWithoutBCReady) && !bcexists {
		return errors.New("Blockchain is not found. Must be created or inited")

	} else if bcexists && utils.StringInSlice(c.Command, disableWithBCReady) {
		return errors.New("Blockchain already exists")
	}

	defer c.Node.DBConn.CloseConnection()

	switch c.Command {
	case "initblockchain":
		return c.commandInitBlockchain()

	case "importblockchain":
		return c.commandImportBlockchain()

	case "restoreblockchain":
		return c.commandRestoreBlockchain()

	case "dumpblockchain":
		return c.commandDumpBlockchain()

	case "exportconsensusconfig":
		return c.commandExportConsensusConfig()

	case "pullupdates":
		return c.commandPullUpdates()

	case "printchain":
		return c.commandPrintChain()

	case "reindexcache":
		return c.commandReindexCache()

	case "getbalance":
		return c.commandGetBalance()

	case "getbalances":
		return c.commandAddressesBalance()

	case "listaddresses":
		return c.forwardCommandToWallet()

	case "createwallet":
		return c.forwardCommandToWallet()

	case "send":
		return c.commandSend()

	case "sql":
		return c.commandSQL()

	case "unapprovedtransactions":
		return c.commandUnapprovedTransactions()

	case "makeblock":
		return c.commandMakeBlock()

	case "dropblock":
		return c.commandDropBlock()

	case "canceltransaction":
		return c.commandCancelTransaction()

	case "addrhistory":
		return c.commandAddressHistory()

	case "showunspent":
		return c.commandShowUnspent()

	case "shownodes":
		return c.commandShowNodes()

	case "addnode":
		return c.commandAddNode()

	case "removenode":
		return c.commandRemoveNode()
	}

	return errors.New("Unknown management command")
}

/*
* Creates node server daemon manager
 */
func (c NodeCLI) createDaemonManager() (*server.NodeDaemon, error) {
	nd := server.NodeDaemon{}

	if !c.Node.BlockchainExist() {
		return nil, errors.New("Blockchain is not found. Must be created or inited")
	}

	nd.ConfigDir = c.ConfigDir
	nd.Logger = c.Logger
	nd.Port = c.Input.Port
	nd.Host = c.Input.Host
	nd.LocalPort = c.Input.LocalPort
	nd.Node = c.Node
	nd.DBProxyAddr = c.Input.DBProxyAddress
	nd.DBAddr = c.Input.Database.GetServerAddress()
	nd.Init()

	return &nd, nil
}

// Execute server management command

func (c NodeCLI) ExecuteManageCommand() error {
	err := c.CreateNode()

	if err != nil {
		return err
	}

	if c.Command == "importandstart" {
		return c.commandImportStartInteractive()

	} else if c.Command == "interactiveautocreate" {
		return c.commandInitIfNeededStartInteractive()
	}
	noddaemon, err := c.createDaemonManager()

	if err != nil {
		return err
	}

	if c.Command == "startnode" {
		return noddaemon.StartServer()

	} else if c.Command == "startintnode" {
		return noddaemon.StartServerInteractive()

	} else if c.Command == "stopnode" {
		return noddaemon.StopServer()

	} else if c.Command == config.Daemonprocesscommandline {
		return noddaemon.DaemonizeServer()

	} else if c.Command == "nodestate" {
		return c.commandShowState(noddaemon)

	}
	return errors.New("Unknown node manage command")
}

// Creates wallet object for operation related to wallets list management
func (c *NodeCLI) getWalletsCLI() (*remoteclient.WalletCLI, error) {
	winput := remoteclient.AppInput{}
	winput.Command = c.Input.Command
	winput.Address = c.Input.Args.Address
	winput.ConfigDir = c.Input.ConfigDir
	winput.NodePort = c.Input.LocalPort
	winput.NodeHost = "localhost"
	winput.Amount = c.Input.Args.Amount
	winput.ToAddress = c.Input.Args.To
	winput.SQL = c.Input.Args.SQL

	if c.Input.Args.From != "" {
		winput.Address = c.Input.Args.From
	}
	//c.Logger.Trace.Println("Running port ", c.AlreadyRunningPort)

	walletscli := remoteclient.WalletCLI{}

	if c.AlreadyRunningPort > 0 {
		winput.NodePort = c.AlreadyRunningPort
		winput.NodeHost = "localhost"
	}

	walletscli.Init(c.Logger, winput)

	walletscli.NodeMode = true

	return &walletscli, nil
}

// Forwards a command to wallet object. This is needed for cases when a node does some
// operation with local wallets
func (c *NodeCLI) forwardCommandToWallet() error {
	walletscli, err := c.getWalletsCLI()

	if err != nil {
		return err
	}
	c.Logger.Trace.Println("Execute command as a client")
	return walletscli.ExecuteCommand()
}

// Create Network Client object. We do this when a node server is running and we need to send
// command to it (indtead of accessing database directly)
func (c *NodeCLI) getLocalNetworkClient() nodeclient.NodeClient {
	nc := *c.Node.NodeClient
	nc.NodeAddress.Port = c.AlreadyRunningPort
	nc.NodeAddress.Host = "localhost"
	return nc
}

// To create new blockchain from scratch
func (c *NodeCLI) commandInitBlockchain() error {
	err := c.Node.CreateBlockchain(c.Input.MinterAddress)

	if err != nil {
		return err
	}
	// some argument are posted with command line. we remember now all of them and will use later
	c.Input.UpdateConfig()

	fmt.Println("Done!")

	return nil
}

// To init blockchain loaded from other node. Is executed for new nodes if blockchain already exists
func (c *NodeCLI) commandImportBlockchain() error {

	if c.Input.Args.NodePort == 0 || c.Input.Args.NodeHost == "" {
		addr := c.Node.ConsensusConfig.GetRandomInitialAddress()

		if addr == nil {
			return errors.New("No address to import from")
		}
		c.Input.Args.NodePort = addr.Port
		c.Input.Args.NodeHost = addr.Host
	}

	alldone, err := c.Node.InitBlockchainFromOther(c.Input.Args.NodeHost, c.Input.Args.NodePort)

	if err != nil {
		return err
	}
	if alldone {
		fmt.Println("Done! ")
	} else {
		fmt.Println("Done! First part of bockchain loaded. Next part will be loaded on background when node started")
	}

	c.Input.UpdateConfig()

	return nil
}

// To restore blockchain from full dump to empty database
func (c *NodeCLI) commandRestoreBlockchain() error {
	if c.Input.Args.DumpFile == "" {
		return errors.New("Dump file name required")
	}
	err := c.Node.RestoreBlockchain(c.Input.Args.DumpFile)

	if err != nil {
		return err
	}
	c.Input.UpdateConfig()
	fmt.Println("Blockchain was inited from DB dump")
	return nil
}

// To create full dump of a blockchain
func (c *NodeCLI) commandDumpBlockchain() error {
	if c.Input.Args.DumpFile == "" {
		return errors.New("Dump file name required")
	}
	err := c.Node.DumpBlockchain(c.Input.Args.DumpFile)

	if err != nil {
		return err
	}
	fmt.Println("Blockchain DB was dumped to a file")
	return nil
}

// Pull updates from all other known nodes
func (c *NodeCLI) commandPullUpdates() error {

	c.Node.GetCommunicationManager().CheckForChangesOnOtherNodes(time.Now().Unix() - 3600)

	fmt.Println("Updates Pull Complete")
	return nil
}

// Print full blockchain
func (c *NodeCLI) commandPrintChain() error {
	bci, err := c.Node.GetBlockChainIterator()

	if err != nil {
		return err
	}

	blocks := []string{}

	for {
		blockfull, err := bci.Next()

		if err != nil {
			return err
		}

		if blockfull == nil {
			fmt.Printf("Somethign went wrong. Next block can not be loaded\n")
			break
		}
		block := blockfull.GetSimpler()

		if c.Input.Args.View == "short" {

			fmt.Printf("===============\n")
			fmt.Printf("Hash: %x\n", block.Hash)
			fmt.Printf("Height: %d, Transactions: %d\n", block.Height, len(block.Transactions)-1)
			fmt.Printf("Prev: %x\n", block.PrevBlockHash)

			fmt.Printf("\n")
		} else if c.Input.Args.View == "shortr" {
			b := fmt.Sprintf("Hash: %x\n", block.Hash)
			b = b + fmt.Sprintf("Height: %d, Transactions: %d\n", block.Height, len(block.Transactions)-1)
			b = b + fmt.Sprintf("Prev: %x\n", block.PrevBlockHash)
			blocks = append(blocks, b)
		} else {
			fmt.Printf("============ Block %x ============\n", block.Hash)
			fmt.Printf("Height: %d\n", block.Height)
			fmt.Printf("Prev. block: %x\n", block.PrevBlockHash)

			for _, tx := range block.Transactions {
				fmt.Println(tx)
			}
			fmt.Printf("\n\n")
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	if c.Input.Args.View == "shortr" {
		for i := len(blocks) - 1; i >= 0; i-- {
			block := blocks[i]
			fmt.Printf("===============\n")
			fmt.Print(block)
			fmt.Printf("\n")
		}
	}

	return nil
}

// Show contents of a cache of unapproved transactions (transactions pool)
func (c *NodeCLI) commandUnapprovedTransactions() error {
	c.Logger.Trace.Println("Show unapproved transactions")
	if c.Input.Args.Clean {
		// clean cache

		return c.Node.GetTransactionsManager().CleanUnapprovedCache()
	}

	total, _ := c.Node.GetTransactionsManager().ForEachUnapprovedTransaction(
		func(txhash, txstr string) error {

			fmt.Println(txstr)

			return nil
		})
	fmt.Printf("\nTotal transactions: %d\n", total)
	return nil
}

// Show all wallets and balances for each of them
func (c *NodeCLI) commandAddressesBalance() error {
	if c.AlreadyRunningPort > 0 {
		// run in wallet mode.
		return c.forwardCommandToWallet()
	}

	walletscli, err := c.getWalletsCLI()

	if err != nil {
		return err
	}
	// get addresses in local wallets
	result := map[string]remoteclient.WalletBalance{}

	for _, address := range walletscli.WalletsObj.GetAddresses() {
		balance, err := c.Node.GetTransactionsManager().GetAddressBalance(address)

		if err != nil {
			return err
		}
		result[string(address)] = balance
	}

	fmt.Println("Balance for all addresses:")
	fmt.Println()

	for address, balance := range result {
		fmt.Printf("%s: %.8f (Approved - %.8f, Pending - %.8f)\n", address, balance.Total, balance.Approved, balance.Pending)
	}

	return nil
}

// Show history for a wallet
func (c *NodeCLI) commandAddressHistory() error {
	if c.AlreadyRunningPort > 0 {
		c.Input.Command = "showhistory"
		// run in wallet mode.
		return c.forwardCommandToWallet()
	}

	result, err := c.Node.NodeBC.GetAddressHistory(c.Input.Args.Address)

	if err != nil {
		return err
	}
	fmt.Println("History of transactions:")
	for _, rec := range result {
		if rec.IOType {
			fmt.Printf("%f\t In from\t%s\n", rec.Value, rec.Address)
		} else {
			fmt.Printf("%f\t Out To  \t%s\n", rec.Value, rec.Address)
		}

	}

	return nil
}

// Show unspent transactions outputs for address
func (c *NodeCLI) commandShowUnspent() error {
	if c.AlreadyRunningPort > 0 {
		// run in wallet mode.
		return c.forwardCommandToWallet()
	}

	balance := float64(0)

	err := c.Node.GetTransactionsManager().ForEachUnspentOutput(c.Input.Args.Address,
		func(fromaddr string, value float64, txID []byte, output int, isbase bool) error {
			fmt.Printf("%f\t from\t%s in transaction %x output #%d\n", value, fromaddr, txID, output)
			balance += value
			return nil
		})

	if err != nil {
		return err
	}

	fmt.Printf("\nBalance - %f\n", balance)

	return nil
}

// Display balance for address
func (c *NodeCLI) commandGetBalance() error {
	if c.AlreadyRunningPort > 0 {
		// run in wallet mode.
		return c.forwardCommandToWallet()
	}

	balance, err := c.Node.GetTransactionsManager().GetAddressBalance(c.Input.Args.Address)

	if err != nil {
		return err
	}

	fmt.Printf("Balance of '%s': \nTotal - %.8f\n", c.Input.Args.Address, balance.Total)
	fmt.Printf("Approved - %.8f\n", balance.Approved)
	fmt.Printf("Pending - %.8f\n", balance.Pending)
	return nil
}

// Send money to other address
func (c *NodeCLI) commandSend() error {
	if c.AlreadyRunningPort > 0 {

		// run in wallet mode.
		return c.forwardCommandToWallet()
	}
	c.Logger.Trace.Println("Send with dirct access to DB ")

	// else, access directtly to the DB

	walletscli, err := c.getWalletsCLI()

	if err != nil {
		return err
	}

	walletobj, err := walletscli.WalletsObj.GetWallet(c.Input.Args.From)

	if err != nil {
		return err
	}

	txid, err := c.Node.Send(walletobj.GetPublicKey(), walletobj.GetPrivateKey(),
		c.Input.Args.To, c.Input.Args.Amount)

	if err != nil {
		return err
	}

	fmt.Printf("Success. New transaction: %x\n", txid)

	return nil
}

// Reindex cache of transactions information
func (c *NodeCLI) commandReindexCache() error {
	info, err := c.Node.GetTransactionsManager().ReindexData()

	if err != nil {
		return err
	}

	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", info["unspentoutputs"])
	return nil
}

// Try to mine a block if there is anough unapproved transactions
func (c *NodeCLI) commandMakeBlock() error {
	block, err := c.Node.TryToMakeBlock([]byte{}, nil)

	if err != nil {
		return err
	}

	if len(block) > 0 {
		fmt.Printf("Done! New block mined with the hash %x.\n", block)
	} else {
		fmt.Printf("Not enough transactions to mine a block.\n")
	}

	return nil
}

// Cancel transaction if it is not yet in a block
func (c *NodeCLI) commandCancelTransaction() error {
	txID, err := hex.DecodeString(c.Input.Args.Transaction)
	if err != nil {
		return err
	}

	err = c.Node.GetTransactionsManager().CancelTransaction(txID, true)

	if err != nil {
		return err
	}

	fmt.Printf("Done!\n")
	fmt.Printf("NOTE. This canceled transaction only from local node. If it was already sent to other nodes, than a transaction still can be completed!\n")

	return nil
}

// Drops last block from the top of blockchain
func (c *NodeCLI) commandDropBlock() error {

	err := c.Node.DropBlock()

	if err != nil {
		return err
	}

	bci, err := c.Node.GetBlockChainIterator()

	if err != nil {
		return err
	}

	blockFull, _ := bci.Next()

	if blockFull == nil {
		return errors.New("This was last block!")
	}
	block := blockFull.GetSimpler()

	fmt.Printf("Done!\n")
	fmt.Printf("============ Last Block %x ============\n", block.Hash)
	fmt.Printf("Height: %d\n", block.Height)
	fmt.Printf("Prev. block: %x\n", block.PrevBlockHash)

	for _, tx := range block.Transactions {
		fmt.Println(tx)
	}
	fmt.Printf("\n\n")

	return nil
}

// Shows server state
func (c *NodeCLI) commandShowState(daemon *server.NodeDaemon) error {
	Runnning, ProcessID, Port, err := daemon.GetServerState()

	fmt.Println("Node Server State:")

	var info nodeclient.ComGetNodeState

	if Runnning {
		fmt.Printf("Server is running. Process: %d, listening on the port %d\n", ProcessID, Port)

		// request state from the node
		nc := c.getLocalNetworkClient()

		info, err = nc.SendGetState()

	} else {
		fmt.Println("Server is not running")
		info, err = c.Node.GetNodeState()
	}

	if err != nil {
		return err
	}

	fmt.Println("Blockchain state:")

	fmt.Printf("  Number of blocks - %d\n", info.BlocksNumber)

	if info.ExpectingBlocksHeight > info.BlocksNumber {
		fmt.Printf("  Loaded %d of %d blocks\n", info.BlocksNumber, info.ExpectingBlocksHeight+1)
	}

	fmt.Printf("  Number of unapproved transactions - %d\n", info.TransactionsCached)

	fmt.Printf("  Number of unspent transactions outputs - %d\n", info.UnspentOutputs)

	return nil
}

// Displays list of nodes (connections)
func (c *NodeCLI) commandShowNodes() error {
	var nodes []net.NodeAddr
	var err error

	if c.AlreadyRunningPort > 0 {
		// connect to node to get nodes list
		nc := c.getLocalNetworkClient()
		nodes, err = nc.SendGetNodes()

		if err != nil {
			return err
		}
	} else {
		nodes = c.Node.NodeNet.GetNodes()
	}
	fmt.Println("Nodes:")

	for _, n := range nodes {
		fmt.Println("  ", n.NodeAddrToString())

	}

	return nil
}

// Add a node to connections
func (c *NodeCLI) commandAddNode() error {
	newaddr := net.NewNodeAddr(c.Input.Args.NodeHost, c.Input.Args.NodePort)

	if c.AlreadyRunningPort > 0 {
		nc := c.getLocalNetworkClient()

		err := nc.SendAddNode(newaddr)

		if err != nil {
			return err
		}
	} else {
		c.Node.AddNodeToKnown(newaddr, false)
	}

	fmt.Println("Success!")

	return nil
}

// Remove a node from connections
func (c *NodeCLI) commandRemoveNode() error {
	remaddr := net.NewNodeAddr(c.Input.Args.NodeHost, c.Input.Args.NodePort)
	fmt.Printf("Remove %s %d", c.Input.Args.NodeHost, c.Input.Args.NodePort)
	fmt.Println(remaddr)

	if c.AlreadyRunningPort > 0 {
		nc := c.getLocalNetworkClient()

		err := nc.SendRemoveNode(remaddr)

		if err != nil {
			return err
		}
	} else {
		c.Node.NodeNet.RemoveNodeFromKnown(remaddr)
	}
	fmt.Println("Success!")

	return nil
}

// Execute new SQL command
func (c *NodeCLI) commandSQL() error {
	if c.AlreadyRunningPort > 0 {

		// run in wallet mode.
		return c.forwardCommandToWallet()
	}
	c.Logger.Trace.Println("Execute SQL with manager tool")

	walletscli, err := c.getWalletsCLI()

	if err != nil {
		return err
	}

	walletobj, err := walletscli.WalletsObj.GetWallet(c.Input.Args.From)

	if err != nil {
		return err
	}

	txid, err := c.Node.SQLTransaction(walletobj.GetPublicKey(), walletobj.GetPrivateKey(), c.Input.Args.SQL)

	if err != nil {
		return err
	}

	if txid == nil {
		fmt.Printf("The query was executed without a transaction\n")
	} else {
		fmt.Printf("Success. New transaction: %x\n", txid)
	}

	return nil
}

// Prepare wallet, import BC and start interactive. If BC exists we just start a server (do nothign before it)
func (c *NodeCLI) commandImportStartInteractive() error {

	if c.Input.Args.LogDestDefault {
		// we always log to stdout for this command
		// if default logging is used (it is files by defaut)
		c.Logger.LogToStdout()
	}

	bcexists := c.Node.BlockchainExist()
	// check if BC exists
	if !bcexists {
		err := c.commandImportBlockchain()

		if err != nil {
			return err
		}
		// check if there is at least 1 wallet . if no, create new one
		walletscli, err := c.getWalletsCLI()

		if err != nil {
			return err
		}

		addresses := walletscli.WalletsObj.GetAddresses()
		// get addresses in local wallets
		if len(addresses) == 0 {
			c.Input.MinterAddress, err = walletscli.WalletsObj.CreateWallet()

			if err != nil {
				return err
			}
		} else {
			c.Input.MinterAddress = addresses[0]
		}

		c.Node.MinterAddress = c.Input.MinterAddress
		c.Input.ProxyKey = c.Input.MinterAddress

		c.setNodeProxyKeys()

		c.Input.UpdateConfig()
	}
	noddaemon, err := c.createDaemonManager()

	if err != nil {
		return err
	}
	return noddaemon.StartServerInteractive()
}
func (c *NodeCLI) commandInitIfNeededStartInteractive() error {

	if c.Input.Args.LogDestDefault {
		// we always log to stdout for this command
		// if default logging is used (it is files by defaut)
		c.Logger.LogToStdout()
	}

	bcexists := c.Node.BlockchainExist()
	// check if BC exists
	if !bcexists {
		// check if there is at least 1 wallet . if no, create new one
		walletscli, err := c.getWalletsCLI()

		if err != nil {
			return err
		}

		addresses := walletscli.WalletsObj.GetAddresses()
		// get addresses in local wallets
		if len(addresses) == 0 {
			c.Input.MinterAddress, err = walletscli.WalletsObj.CreateWallet()

			if err != nil {
				return err
			}
		} else {
			c.Input.MinterAddress = addresses[0]
		}

		// init BC with this address as a minter

		err = c.commandInitBlockchain()

		if err != nil {
			return err
		}
		c.Node.MinterAddress = c.Input.MinterAddress
		c.Input.ProxyKey = c.Input.MinterAddress

		c.setNodeProxyKeys()

		c.Input.UpdateConfig()
	}
	noddaemon, err := c.createDaemonManager()

	if err != nil {
		return err
	}
	return noddaemon.StartServerInteractive()
}

// Export consensus config to given destination. This can be used for distribution of an app
func (c *NodeCLI) commandExportConsensusConfig() error {

	if c.Input.Port == 0 &&
		(c.Input.Args.DefaultAddresses == "" || c.Input.Args.DefaultAddresses == "own") &&
		len(c.Node.ConsensusConfig.InitNodesAddreses) == 0 {
		return errors.New("No known address to use as a default. Set network address for this node first")
	}

	ownAddres := net.NewNodeAddr(c.Input.Host, c.Input.Port)

	c.Logger.Trace.Printf("Export Consensus Config file. Own address %s", ownAddres.NodeAddrToString())

	if c.Input.Args.DestinationFile == "" {
		return errors.New("Destination file path missed")
	}

	return c.Node.ConsensusConfig.ExportToFile(
		c.Input.Args.DestinationFile,
		c.Input.Args.DefaultAddresses,
		c.Input.Args.AppName,
		ownAddres.NodeAddrToString())
}
