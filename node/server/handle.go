package server

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/lib/nodeclient"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/blockchain"
	"github.com/gelembjuk/oursql/node/nodemanager"
	"github.com/gelembjuk/oursql/node/structures"
	"github.com/gelembjuk/oursql/node/transactions"
)

type NodeServerRequest struct {
	Node              *nodemanager.Node
	S                 *NodeServer
	Request           []byte
	RequestIP         string
	Logger            *utils.LoggerMan
	HasResponse       bool
	Response          []byte
	NodeAuthStrIsGood bool
	SessID            string
}

func (s *NodeServerRequest) Init() {
	s.HasResponse = false
	s.Response = nil
}

// Reads and parses request from network data
func (s *NodeServerRequest) parseRequestData(payload interface{}) error {
	var buff bytes.Buffer

	buff.Write(s.Request)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(payload)

	if err != nil {
		return errors.New("Parse request: " + err.Error())
	}

	return nil
}

// Find and return the list of unspent transactions
func (s *NodeServerRequest) handleGetUnspent() error {
	s.HasResponse = true

	var payload nodeclient.ComGetUnspentTransactions

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	result := nodeclient.ComUnspentTransactions{}

	result.LastBlock, err = s.Node.NodeBC.GetTopBlockHash()

	if err != nil {
		return err
	}

	err = s.Node.GetTransactionsManager().ForEachUnspentOutput(payload.Address,
		func(fromaddr string, value float64, txID []byte, output int, isbase bool) error {
			ut := nodeclient.ComUnspentTransaction{}
			ut.Amount = value
			ut.TXID = txID
			ut.Vout = output
			ut.From = fromaddr
			ut.IsBase = isbase

			result.Transactions = append(result.Transactions, ut)
			return nil
		})

	if err != nil {
		return err
	}

	s.Response, err = net.GobEncode(result)

	if err != nil {
		return err
	}
	s.Logger.Trace.Printf("Return %d unspent transactions for %s\n", len(result.Transactions), payload.Address)
	return nil
}

// Find and return  history of transactions for wallet address
func (s *NodeServerRequest) handleGetHistory() error {
	s.HasResponse = true

	var payload nodeclient.ComGetHistoryTransactions

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	result := []nodeclient.ComHistoryTransaction{}

	history, err := s.Node.NodeBC.GetAddressHistory(payload.Address)

	if err != nil {
		return err
	}

	for _, t := range history {
		ut := nodeclient.ComHistoryTransaction{}
		ut.Amount = t.Value
		ut.IOType = t.IOType
		ut.TXID = t.TXID

		if t.IOType {
			ut.From = t.Address
		} else {
			ut.To = t.Address
		}
		result = append(result, ut)
	}

	s.Response, err = net.GobEncode(result)

	if err != nil {
		return err
	}
	s.Logger.Trace.Printf("Return %d history records for %s\n", len(result), payload.Address)
	return nil
}

// Balance for address. Complex balance
func (s *NodeServerRequest) handleGetBalance() error {
	s.HasResponse = true

	var payload nodeclient.ComGetWalletBalance

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	balance := nodeclient.ComWalletBalance{}

	balancen, err := s.Node.GetTransactionsManager().GetAddressBalance(payload.Address)

	if err != nil {
		return err
	}
	// we copy with this way because the structure WalletBalance in not known on nodeclient
	balance.Total = balancen.Total
	balance.Approved = balancen.Approved
	balance.Pending = balancen.Pending

	s.Response, err = net.GobEncode(balance)

	if err != nil {
		return err
	}
	s.Logger.Trace.Printf("Return balance for %s. %.8f, %.8f, %.8f", payload.Address, balance.Total, balance.Approved, balance.Pending)
	return nil
}

// Accepts new transaction. Adds to the list of unapproved. then try to build a block
// This is the request from wallet. Not from other node.
func (s *NodeServerRequest) handleTxFull() error {
	s.HasResponse = true

	var payload nodeclient.ComNewTransaction

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}
	TX := structures.Transaction{}
	TX.DeserializeTransaction(payload.TX)

	err = s.Node.GetTransactionsManager().ReceivedNewTransaction(&TX)

	if err != nil {
		return errors.New(fmt.Sprintf("Transaction accepting error: %s", err.Error()))
	}

	s.Logger.Trace.Printf("Acceppted new transaction from %s\n", payload.Address)

	// send internal command to try to mine new block

	s.S.TryToMakeNewBlock(TX.ID)

	s.Response, err = net.GobEncode(payload.TX)

	if err != nil {
		return errors.New(fmt.Sprintf("TXFull Response Error: %s", err.Error()))
	}
	return nil
}

// Accepts new transaction data. It is prepared transaction without signatures
// Signatures are received too. Complete TX must be constructed and verified.
// If all is ok TXt is added to unapproved and ID returned
func (s *NodeServerRequest) handleTxData() error {
	s.HasResponse = true

	var payload nodeclient.ComNewTransactionData

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	TX, err := s.Node.GetTransactionsManager().ReceivedNewTransactionData(payload.TX, payload.Signatures)

	if err != nil {
		return errors.New(fmt.Sprintf("Transaction accepting error: %s", err.Error()))
	}

	s.Logger.Trace.Printf("Acceppted new transaction from %s\n", payload.Address)

	// send internal command to try to mine new block

	s.S.TryToMakeNewBlock(TX.ID)

	s.Response, err = net.GobEncode(TX.ID)

	if err != nil {
		return errors.New(fmt.Sprintf("TXFull Response Error: %s", err.Error()))
	}
	return nil
}

/*
* Request for new transaction from light client. Builds a transaction without sign.
* Returns also list of previous transactions selected for input. it is used for signature on client side
 */
func (s *NodeServerRequest) handleTxRequest() error {
	s.HasResponse = true

	var payload nodeclient.ComRequestTransaction

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	result := nodeclient.ComRequestTransactionData{}

	TXBytes, DataToSign, err := s.Node.GetTransactionsManager().
		PrepareNewTransaction(payload.PubKey, payload.To, payload.Amount)

	if err != nil {
		return err
	}

	result.DataToSign = DataToSign
	result.TX = TXBytes

	s.Response, err = net.GobEncode(result)

	if err != nil {
		return err
	}

	return nil
}

/*
* Handle request from a new node where a blockchain is not yet inted.
* This s ed to get the first part of blocks to init local blockchain DB
 */
func (s *NodeServerRequest) handleGetFirstBlocks() error {
	s.HasResponse = true

	result := nodeclient.ComGetFirstBlocksData{}

	blocks, height, err := s.Node.NodeBC.GetBCManager().GetFirstBlocks(10)

	if err != nil {
		return err
	}

	result.Blocks = [][]byte{}
	result.Height = height

	for _, block := range blocks {
		blockdata, err := block.Serialize()

		if err != nil {
			return err
		}
		result.Blocks = append(result.Blocks, blockdata)
	}

	s.Response, err = net.GobEncode(result)

	if err != nil {
		return err
	}

	s.Logger.Trace.Printf("Return first %d blocks\n", len(blocks))
	return nil
}

// Received the lst of nodes from some other node. add missed nodes to own nodes list

func (s *NodeServerRequest) handleAddr() error {
	var payload []net.NodeAddr
	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}
	addednodes := []net.NodeAddr{}

	//s.Logger.Trace.Printf("SessID: %s . Received nodes %s", s.SessID, payload)

	for _, node := range payload {
		//s.Logger.Trace.Printf("SessID: %s . node %s", s.SessID, node.NodeAddrToString())
		if s.S.Node.NodeNet.AddNodeToKnown(node) {
			addednodes = append(addednodes, node)
			//s.Logger.Trace.Printf("SessID: %s . node appended %s", s.SessID, node.NodeAddrToString())
		}
	}

	//s.Logger.Trace.Printf("SessID: %s . There are %d known nodes now!", s.SessID, len(s.Node.NodeNet.Nodes))
	//s.Logger.Trace.Printf("SessID: %s . Send version to %d new nodes", s.SessID, len(addednodes))

	if len(addednodes) > 0 {
		// send own version to all new found nodes. maybe they have some more blocks
		// and they will add me to known nodes after this
		s.Node.SendVersionToNodes(addednodes)
	}

	return nil
}

// Block received from other node
func (s *NodeServerRequest) handleBlock() error {
	var payload nodeclient.ComBlock
	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	blockstate, addstate, block, err := s.Node.ReceivedFullBlockFromOtherNode(payload.Block)
	s.Logger.Trace.Printf("adding new block %d, %d", blockstate, addstate)
	// state of this adding we don't check. not interesting in this place
	if err != nil {
		return err
	}

	if blockstate == 0 {
		s.Logger.Trace.Printf("send block to all ")
		// block was added, now we can send it to all other nodes.
		s.Node.SendBlockToAll(block, payload.AddrFrom)
	}
	// this is the list of hashes some node posted before. If there are yes some data then try to get that blocks.
	s.Logger.Trace.Printf("check count blocks left %d ", s.S.Transit.GetBlocksCount(payload.AddrFrom))
	if s.S.Transit.GetBlocksCount(payload.AddrFrom) > 0 {
		// get next block. continue to get next block if nothing is sent
		for {
			blockdata, err := s.S.Transit.ShiftNextBlock(payload.AddrFrom)

			if err != nil {
				s.Logger.Trace.Printf("Request new block failed %s ", err.Error())
				return err
			}

			blockstate, err := s.Node.ReceivedBlockFromOtherNode(payload.AddrFrom, blockdata)

			if err != nil {
				return err
			}

			if blockstate == 0 {
				// we requested one block info. stop for now
				break
			}

			if blockstate == 2 {
				// previous block is not in the blockchain. no sense to check next blocks in this list
				s.S.Transit.CleanBlocks(payload.AddrFrom)

				// request from a node blocks down to this first block
				bs := &structures.BlockShort{}
				err := bs.DeserializeBlock(blockdata)

				if err != nil {
					return err
				}
				// get blocks down stargin from previous for the first in given list
				s.Node.NodeClient.SendGetBlocks(payload.AddrFrom, bs.PrevBlockHash)
			}

			if s.S.Transit.GetBlocksCount(payload.AddrFrom) == 0 {
				break
			}
		}
	}
	s.Logger.Trace.Printf("check if try to make new %d , %d ", addstate, blockchain.BCBAddState_addedToParallelTop)
	if addstate == blockchain.BCBAddState_addedToParallelTop {
		// maybe some transactiosn become unapproved now. try to make new block from them on top of new chain
		s.S.TryToMakeNewBlock([]byte{1})
	}
	s.Node.CheckAddressKnown(payload.AddrFrom)

	return nil
}

/*
* Other node posted info about new blocks or new transactions
* This contains only a hash of a block or ID of a transaction
* If such block or transaction is not yet present , then request for full info about it
 */
func (s *NodeServerRequest) handleInv() error {
	var payload nodeclient.ComInv

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	s.Logger.Trace.Printf("SessID: %s . Recevied inventory with %d %s\n", s.SessID, len(payload.Items), payload.Type)

	if payload.Type == "block" {

		s.S.Transit.AddBlocks(payload.AddrFrom, payload.Items)

		for {

			blockdata, err := s.S.Transit.ShiftNextBlock(payload.AddrFrom)

			if err != nil {
				return err
			}

			blockstate, err := s.Node.ReceivedBlockFromOtherNode(payload.AddrFrom, blockdata)

			if err != nil {
				return err
			}

			if blockstate == 0 {
				// we requested one block info. stop for now
				break
			}

			if blockstate == 2 {
				// previous block is not in the blockchain. no sense to check next blocks in this list
				s.S.Transit.CleanBlocks(payload.AddrFrom)

				// request from a node blocks down to this first block
				bs := &structures.BlockShort{}
				err := bs.DeserializeBlock(blockdata)

				if err != nil {
					return err
				}
				// get blocks down stargin from previous for the first in given list
				s.Node.NodeClient.SendGetBlocks(payload.AddrFrom, bs.PrevBlockHash)
			}

			if s.S.Transit.GetBlocksCount(payload.AddrFrom) == 0 {
				break
			}
		}

	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		s.Logger.Trace.Printf("Check if TX exists %x\n", txID)

		tx, err := s.Node.GetTransactionsManager().GetIfExists(txID)

		if tx == nil && err == nil {
			// not exists
			s.Logger.Trace.Printf("Not exist. Request it\n")
			s.Node.NodeClient.SendGetData(payload.AddrFrom, "tx", txID)
		}
	}
	s.Node.CheckAddressKnown(payload.AddrFrom)

	return nil
}

/*
* Request to get list of blocks hashes .
* It can contain a starting block hash to return data from it
* If no that starting hash, then data from a top are returned
 */
func (s *NodeServerRequest) handleGetBlocks() error {
	var payload nodeclient.ComGetBlocks

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	blocks := s.Node.NodeBC.GetBCManager().GetBlocksShortInfo(payload.StartFrom, 1000)

	s.Logger.Trace.Printf("Loaded %d block hashes", len(blocks))

	data := [][]byte{}

	for i := len(blocks) - 1; i >= 0; i-- {
		bdata, _ := blocks[i].Serialize()
		data = append(data, bdata)
		s.Logger.Trace.Printf("Block: %x", blocks[i].Hash)
	}
	s.Node.CheckAddressKnown(payload.AddrFrom)
	return s.Node.NodeClient.SendInv(payload.AddrFrom, "block", data)
}

/*
* Request to get all blocks up to given block.
* Nodes use it to load missed blocks from other node.
* If requested bock is not found in BC then TOP blocks are returned
 */
func (s *NodeServerRequest) handleGetBlocksUpper() error {
	var payload nodeclient.ComGetBlocks

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	s.Logger.Trace.Printf("Get blocks after %x", payload.StartFrom)

	blocks, err := s.Node.NodeBC.GetBlocksAfter(payload.StartFrom)

	if err != nil {
		return err
	}

	if blocks == nil {
		s.Logger.Trace.Printf("Nothing found after %x. Return top of the blockchain", payload.StartFrom)

		blocks = s.Node.NodeBC.GetBCManager().GetBlocksShortInfo([]byte{}, 1000)
	}

	s.Logger.Trace.Printf("Loaded %d block hashes", len(blocks))

	data := [][]byte{}

	for i := len(blocks) - 1; i >= 0; i-- {
		bdata, _ := blocks[i].Serialize()
		data = append(data, bdata)
		s.Logger.Trace.Printf("Block: %x", blocks[i].Hash)
	}

	s.Node.CheckAddressKnown(payload.AddrFrom)

	return s.Node.NodeClient.SendInv(payload.AddrFrom, "block", data)
}

/*
* Response on request to get full body of a block or transaction
 */
func (s *NodeServerRequest) handleGetData() error {
	var payload nodeclient.ComGetData

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	s.Logger.Trace.Printf("SessID: %s . Data Requested of type %s, id %x\n", s.SessID, payload.Type, payload.ID)

	if payload.Type == "block" {

		block, err := s.Node.NodeBC.GetBlock([]byte(payload.ID))
		if err != nil {
			return err
		}

		bs, err := block.Serialize()

		if err == nil {
			s.Node.NodeClient.SendBlock(payload.AddrFrom, bs)
		}

	}

	if payload.Type == "tx" {

		if txe, err := s.Node.GetTransactionsManager().GetIfUnapprovedExists(payload.ID); err == nil && txe != nil {

			s.Logger.Trace.Printf("Return transaction with ID %x to %s\n", payload.ID, payload.AddrFrom.NodeAddrToString())
			// exists
			txser, err := txe.Serialize()

			if err != nil {
				return err
			}

			s.Node.NodeClient.SendTx(payload.AddrFrom, txser)

		}
	}

	s.Node.CheckAddressKnown(payload.AddrFrom)

	return nil
}

/*
* Handle new transaction. Verify it before doing something (verify is done in the NodeTX object)
* This is transaction received from other node. We expect that other node aready posted it to all other
* Here we have a choice. Or we also send it to all other or not.
* For now we don't send it to all other
 */
func (s *NodeServerRequest) handleTx() error {
	var payload nodeclient.ComTx

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	txData := payload.Transaction
	tx := structures.Transaction{}
	err = tx.DeserializeTransaction(txData)

	if err != nil {
		return err
	}

	if txe, err := s.Node.GetTransactionsManager().GetIfExists(tx.ID); err == nil && txe != nil {
		s.Logger.Trace.Printf("Received transaction. It already exists: %x ", tx.ID)
		// exists , nothing to do, it was already processed before
		return nil
	}
	s.Logger.Trace.Printf("Received transaction. It does not exists: %x ", tx.ID)
	// this will also verify a transaction
	err = s.Node.GetTransactionsManager().ReceivedNewTransaction(&tx)

	if err != nil {
		// if error is because some input transaction is not found, then request it and after it this TX again
		s.Logger.Trace.Println("Error ", err.Error())

		if err, ok := err.(*transactions.TXVerifyError); ok {
			s.Logger.Trace.Println("Custom errro of kind ", err.GetKind())

			if err.GetKind() == transactions.TXVerifyErrorNoInput {
				/*
					* we will not do somethign in this case. If no base TX that is not yet approved we wil ignore it
					* previous TX must exist on a node that sent this TX, so, let it complete this work abd build a block
					* This case is possible if a node was not online when previous TX was created
					s.Logger.Trace.Printf("Request another 2 TX %x , %x", err.TX, tx.ID)
					s.Node.NodeClient.SendGetData(payload.AddFrom, "tx", err.TX)
					time.Sleep(1 * time.Second) // wait to get a chance to return that TX
					// TODO we need to be able to request more TX in order in single request
					s.Node.NodeClient.SendGetData(payload.AddFrom, "tx", tx.ID)
					return nil*/

				// TODO in future we can createsomethign more start here. Like, get TX with all previous TXs that are not approved yet
			}

		}
		return err
	}

	// send this transaction to all other nodes
	// TODO
	// maybe we should not send transaction here to all other nodes.
	// this node should try to make a block first.

	// try to mine new block. don't send the transaction to other nodes after block make attempt
	s.S.TryToMakeNewBlock([]byte{0})

	return nil
}

/*
* Process version command. Other node sends own address and index of top block.
* This node checks if index is bogger then request for a rest of blocks. If index is less
* then sends own version command and that node will request for blocks
 */
func (s *NodeServerRequest) handleVersion() error {
	var payload nodeclient.ComVersion

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	topHash, myBestHeight, err := s.Node.NodeBC.GetBCManager().GetState()

	if err != nil {
		return err
	}

	if payload.AddrFrom.Host == "localhost" {
		payload.AddrFrom.Host = s.RequestIP
	}

	s.Logger.Trace.Printf("Received version from %s. Their heigh %d, our heigh %d\n",
		payload.AddrFrom.NodeAddrToString(), payload.BestHeight, myBestHeight)

	foreignerBestHeight := payload.BestHeight

	if myBestHeight < foreignerBestHeight {
		s.Logger.Trace.Printf("Request blocks from %s\n", payload.AddrFrom.NodeAddrToString())

		if foreignerBestHeight > s.S.Transit.MaxKnownHeigh {
			s.S.Transit.MaxKnownHeigh = foreignerBestHeight
		}

		s.Node.NodeClient.SendGetBlocksUpper(payload.AddrFrom, topHash)

	} else if myBestHeight > foreignerBestHeight {
		s.Logger.Trace.Printf("Send my version back to %s\n", payload.AddrFrom.NodeAddrToString())

		s.Node.NodeClient.SendVersion(payload.AddrFrom, myBestHeight)
	} else {
		s.Logger.Trace.Printf("Teir blockchain is same as my for %s\n", payload.AddrFrom.NodeAddrToString())
	}

	s.S.Node.CheckAddressKnown(payload.AddrFrom)

	return nil
}

// Returns list of nodes from contacts on this node

func (s *NodeServerRequest) handleGetNodes() error {
	s.HasResponse = true

	nodes := s.S.Node.NodeNet.GetNodes()

	s.Logger.Trace.Printf("Return %d nodes\n", len(nodes))

	var err error

	s.Response, err = net.GobEncode(&nodes)

	if err != nil {
		return err
	}
	return nil
}

// Add new node to list of nodes
func (s *NodeServerRequest) handleAddNode() error {
	if !s.NodeAuthStrIsGood {
		return errors.New("Local Network Auth is required")
	}

	s.HasResponse = true

	var payload nodeclient.ComManageNode

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	s.S.Node.AddNodeToKnown(payload.Node, true)

	s.Response = []byte{}

	return nil
}

// Remove node from list of nodes
func (s *NodeServerRequest) handleRemoveNode() error {
	if !s.NodeAuthStrIsGood {
		return errors.New("Local Network Auth is required")
	}

	s.HasResponse = true

	var payload nodeclient.ComManageNode

	err := s.parseRequestData(&payload)

	if err != nil {
		return err
	}

	s.S.Node.NodeNet.RemoveNodeFromKnown(payload.Node)

	s.Logger.Trace.Printf("Removed node %s\n", payload.Node.NodeAddrToString())
	s.Logger.Trace.Println(s.S.Node.NodeNet.Nodes)

	s.Response = []byte{}

	return nil
}

// Return node state, including pending blocks to load
func (s *NodeServerRequest) handleGetState() error {
	if !s.NodeAuthStrIsGood {
		return errors.New("Local Network Auth is required")
	}

	s.HasResponse = true

	info, err := s.Node.GetNodeState()

	if err != nil {
		return err
	}

	info.ExpectingBlocksHeight = s.S.Transit.MaxKnownHeigh

	s.Response, err = net.GobEncode(&info)

	if err != nil {
		return err
	}
	return nil
}
