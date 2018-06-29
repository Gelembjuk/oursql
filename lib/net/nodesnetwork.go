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
	Logger  *utils.LoggerMan
	Nodes   []NodeAddr
	Storage NodeNetworkStorage
	lock    *sync.Mutex
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
