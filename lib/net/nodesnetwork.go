package net

import (
	"sync"

	"github.com/gelembjuk/oursql/lib/utils"
)

// INterface for extra storage for a nodes.
// TODO
// This is not used yet
type NodeNetworkStorage interface {
	GetNodes() ([]NodeAddr, error)
	AddNodeToKnown(addr NodeAddr)
	RemoveNodeFromKnown(addr NodeAddr)
	GetCountOfKnownNodes() (int, error)
}

// This manages list of known nodes by a node
type NodeNetwork struct {
	Logger                 *utils.LoggerMan
	Nodes                  []NodeAddr
	hadInputConnects       bool
	hadRecentInputConnects bool
	Storage                NodeNetworkStorage
	lock                   *sync.Mutex
}

type NodesListJSON struct {
	Nodes   []NodeAddr
	Genesis string
}

// Init nodes network object
func (n *NodeNetwork) Init() {
	n.lock = &sync.Mutex{}
}

// Set extra storage for a nodes
func (n *NodeNetwork) SetExtraManager(storage NodeNetworkStorage) {
	n.Storage = storage
}

// Loads list of nodes from storage
func (n *NodeNetwork) LoadNodes() error {
	if n.Storage == nil {
		return nil
	}

	n.lock.Lock()
	defer n.lock.Unlock()

	nodes, err := n.Storage.GetNodes()

	if err != nil {
		return err
	}

	for _, node := range nodes {
		n.Nodes = append(n.Nodes, node)
	}

	return nil
}

// Set nodes list. This can be used to do initial nodes loading from  config or so
func (n *NodeNetwork) SetNodes(nodes []NodeAddr, replace bool) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if replace {
		n.Nodes = nodes
	} else {
		n.Nodes = append(n.Nodes, nodes...)
	}

	if n.Storage != nil {
		// remember what is not yet remembered
		for _, node := range nodes {
			n.Storage.AddNodeToKnown(node)
		}
	}
}

func (n *NodeNetwork) GetNodes() []NodeAddr {
	return n.Nodes
}

// Returns number of known nodes
func (n *NodeNetwork) GetCountOfKnownNodes() int {
	l := len(n.Nodes)

	return l
}

// Check if node address is known
func (n *NodeNetwork) CheckIsKnown(addr NodeAddr) bool {
	exists := false

	for _, node := range n.Nodes {
		if node.CompareToAddress(addr) {
			exists = true
			break
		}
	}

	return exists
}

// Action on input connection from a node. We need to remember this node
// It is needed to know there are input connects from other nodes
func (n *NodeNetwork) InputConnectFromNode(addr NodeAddr) {
	for i, node := range n.Nodes {
		if node.CompareToAddress(addr) {
			n.Nodes[i].SuccessIncomeConnections = n.Nodes[i].SuccessIncomeConnections + 1
			break
		}
	}
	n.hadInputConnects = true
	n.hadRecentInputConnects = true
}

// Sets input connects marker to false to check if there will be new input connects
func (n *NodeNetwork) StartNewSessionForInputConnects() {
	n.hadRecentInputConnects = false
}

// Check if there were recent input connects
func (n *NodeNetwork) CheckHadInputConnects() bool {
	return n.hadRecentInputConnects
}

// Get list of nodes in short format
func (n *NodeNetwork) GetNodesToExport() (list []NodeAddrShort) {
	list = []NodeAddrShort{}

	for _, node := range n.Nodes {
		list = append(list, node.GetShortFormat())
	}
	return
}

/*
* Checks if a node exists in list of known nodes and adds it if no
* Returns true if was added
 */
func (n *NodeNetwork) AddNodeToKnown(addr NodeAddr) bool {
	n.lock.Lock()
	defer n.lock.Unlock()

	exists := false

	for _, node := range n.Nodes {
		if node.CompareToAddress(addr) {
			exists = true
			break
		}
	}
	if !exists {
		n.Nodes = append(n.Nodes, addr)
	}

	if n.Storage != nil {
		n.Storage.AddNodeToKnown(addr)
	}

	return !exists
}

// Removes a node from known
func (n *NodeNetwork) RemoveNodeFromKnown(addr NodeAddr) {
	n.lock.Lock()
	defer n.lock.Unlock()

	updatedlist := []NodeAddr{}

	for _, node := range n.Nodes {
		if !node.CompareToAddress(addr) {
			updatedlist = append(updatedlist, node)
		}
	}

	n.Nodes = updatedlist

	if n.Storage != nil {
		n.Storage.RemoveNodeFromKnown(addr)
	}
}

// Checks nodes rendomly and returns first found node that is accesible
// It check if a node was ever connected from this place
func (n NodeNetwork) GetConnecttionVerifiedNodeAddr() *NodeAddr {
	//n.Logger.Trace.Printf("Currently there are %d nodes", len(n.Nodes))

	if len(n.Nodes) == 0 {
		return nil
	}

	rng := utils.MakeRandomRange(0, len(n.Nodes)-1)

	var i int

	for _, i = range rng {
		node := n.Nodes[i]

		if node.SuccessConnections > 0 {
			return &node
		}
	}
	return &n.Nodes[i]
}

// Same as GetConnecttionVerifiedNodeAddr but returns all verified nodes  or limited list if requested
func (n NodeNetwork) GetConnecttionVerifiedNodeAddresses(limit int) []*NodeAddr {
	nodes := []*NodeAddr{}

	if len(n.Nodes) == 0 {
		return nodes
	}

	rng := utils.MakeRandomRange(0, len(n.Nodes)-1)

	var i int

	for _, i = range rng {
		node := n.Nodes[i]

		if node.SuccessConnections > 0 {
			nodes = append(nodes, &node)

			if limit > 0 && len(nodes) >= limit {
				return nodes
			}
		}
	}
	if len(nodes) == 0 {
		for _, n := range n.Nodes {
			nodes = append(nodes, &n)
		}
	}
	return nodes
}

// Call this when network operation with some node failed.
// It will analise error and do some actios to remember state of this node
func (n *NodeNetwork) HookNeworkOperationResult(err error, nodeindex int) {
	if err == nil {
		n.Nodes[nodeindex].ReportSuccessConn()
		return
	}
	if errv, ok := err.(*NetworkError); ok {
		if errv.WasConnFailure() {
			n.Nodes[nodeindex].ReportFailedConn()
		}
	}

}

// Same as HookNeworkOperationResult but finds a node by address, not by index
func (n *NodeNetwork) HookNeworkOperationResultForNode(err error, nodeU *NodeAddr) {
	for i, node := range n.Nodes {
		if node.CompareToAddress(*nodeU) {
			n.HookNeworkOperationResult(err, i)
			break
		}
	}
}
