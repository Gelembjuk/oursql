package nodemanager

import (
	"github.com/gelembjuk/oursql/lib"
	"github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/lib/utils"

	"github.com/gelembjuk/oursql/node/structures"
)

type communicationManager struct {
	logger *utils.LoggerMan
	node   *Node
}

// Send own version to all known nodes

func (n communicationManager) sendVersionToNodes(nodes []net.NodeAddr, bestHeight int) {

	if len(nodes) == 0 {
		nodes = n.node.NodeNet.Nodes
	}

	for _, node := range nodes {
		if node.CompareToAddress(n.node.NodeClient.NodeAddress) {
			continue
		}
		n.node.NodeClient.SendVersion(node, bestHeight)
	}
}

// Send transaction to all known nodes. This wil send only hash and node hash to check if hash exists or no
func (n *communicationManager) sendTransactionToAll(tx *structures.Transaction) {
	n.logger.Trace.Printf("Send transaction to %d nodes", len(n.node.NodeNet.Nodes))

	for i, node := range n.node.NodeNet.Nodes {
		if node.CompareToAddress(n.node.NodeClient.NodeAddress) {
			continue
		}
		n.logger.Trace.Printf("Send TX %x to %s", tx.GetID(), node.NodeAddrToString())
		err := n.node.NodeClient.SendInv(node, "tx", [][]byte{tx.GetID()})
		n.node.NodeNet.HookNeworkOperationResult(err, i) // to know if this node is available
	}
}

// Send block to all known nodes
// This is used in case when new block was received from other node or
// created by this node. We will notify our network about new block
// But not send full block, only hash and previous hash. So, other can copy it
// Address from where we get it will be skipped
func (n *communicationManager) SendBlockToAll(newBlock *structures.Block, skipaddr net.NodeAddr) {
	for i, node := range n.node.NodeNet.Nodes {
		if node.CompareToAddress(n.node.NodeClient.NodeAddress) {
			continue
		}
		blockshortdata, err := newBlock.GetShortCopy().Serialize()
		if err == nil {
			errc := n.node.NodeClient.SendInv(node, "block", [][]byte{blockshortdata})
			n.node.NodeNet.HookNeworkOperationResult(errc, i) // to know if this node is available
		}
	}
}

// Check for updates on other nodes
func (n *communicationManager) CheckForChangesOnOtherNodes(lastCheckTime int64) error {

	// get max 10 random nodes where connection was success before
	nodes := n.node.NodeNet.GetConnecttionVerifiedNodeAddresses(10)

	n.logger.Trace.Printf("Commun Man: check from %d nodes", len(nodes))

	// get local blockchain state to send request to other

	myBestHeight, topHashes, err := n.node.NodeBC.GetBCTopState(5)

	if err != nil {
		return err
	}

	for _, node := range nodes {
		n.logger.Trace.Printf("Check node %s", node.NodeAddrToString())
		if node.CompareToAddress(n.node.NodeClient.NodeAddress) {
			continue
		}

		n.pullUpdatesFromNode(node, lastCheckTime, topHashes, myBestHeight)
	}
	return nil
}

// Load updates from a node
func (n *communicationManager) pullUpdatesFromNode(node *net.NodeAddr, lastCheckTime int64, topHashes [][]byte, myBestHeight int) error {
	result, err := n.node.NodeClient.SendGetUpdates(*node, lastCheckTime, myBestHeight, topHashes)

	if err != nil {
		n.node.NodeNet.HookNeworkOperationResultForNode(err, node)
		return err
	}

	err = n.processBlocksFromOtherNode(node, result.Blocks)

	if err != nil {
		return err
	}

	err = n.processTransactionsFromPoolOnOtherNode(node, result.TransactionsInPool)

	if err != nil {
		return err
	}

	return nil
}

// Process blocks list received from other node
func (n *communicationManager) processBlocksFromOtherNode(node *net.NodeAddr, blocks [][]byte) error {
	for _, bsdata := range blocks {
		bs, err := structures.NewBlockShortFromBytes(bsdata)

		if err != nil {
			n.logger.Error.Printf("Error when deserialize block info %s", err.Error())
			continue
		}
		// check if block exists
		blockstate, err := n.node.NodeBC.CheckBlockState(bs.Hash, bs.PrevBlockHash)

		if err != nil {
			n.logger.Error.Printf("Error when check block state %s", err.Error())
			continue
		}

		if blockstate == 2 {
			// previous hash is not found. all next blocks will also be missed
			// we don't have good way to process this case
			// we need to go more deeper with requeting of blocks.
			// TODO
			// no solution yet.
			n.logger.Trace.Printf("Previous block not found: %x . Exit blocks loop", bs.PrevBlockHash)
			return nil
		}

		if blockstate == 0 {
			// in this case we can request this block full info

			result, err := n.node.NodeClient.SendGetBlock(*node, bs.Hash)

			if err != nil {
				n.logger.Error.Printf("Error when reuest block body %s", err.Error())
				continue
			}

			n.logger.Trace.Printf("Success loaded block %x", bs.Hash)

			blockstate, _, _, err := n.node.ReceivedFullBlockFromOtherNode(result.Block)

			n.logger.Trace.Printf("adding new block ith state %d", blockstate)
			// state of this adding we don't check. not interesting in this place
			if err != nil {
				n.logger.Error.Printf("Error when adding block body %s", err.Error())
				continue
			}

		}
	}
	return nil
}

// Process blocks list received from other node
func (n *communicationManager) processTransactionsFromPoolOnOtherNode(node *net.NodeAddr, transactions [][]byte) error {
	n.logger.Trace.Printf("Check %d transactions from remote pool", len(transactions))

	if len(transactions) == 0 {
		n.logger.Trace.Printf("Nothing to check")
		return nil
	}

	for _, txID := range transactions {
		// if not exist , request for full body of a TX and add to a pool
		if txe, err := n.node.GetTransactionsManager().GetIfExists(txID); err == nil && txe != nil {
			n.logger.Trace.Printf("TX already exists: %x ", txID)
			continue
		}
		// request this TX and add to the pool
		result, err := n.node.NodeClient.SendGetTransaction(*node, txID)

		if err != nil {
			n.logger.Error.Printf("Error when request for TX from other node %s", err.Error())
			continue
		}
		tx, err := structures.DeserializeTransaction(result.Transaction)

		if err != nil {
			n.logger.Error.Printf("Error when deserialize TX %s", err.Error())
			continue
		}
		n.node.getBlockMakeManager().AddTransactionToPool(tx, lib.TXFlagsExecute)
	}
	return nil
}
