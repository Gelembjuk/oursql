package nodemanager

import (
	"github.com/gelembjuk/oursql/lib/net"
)

type NodesListStorage struct {
	DBConn    *Database
	SessionID string
}

func (s NodesListStorage) GetNodes() ([]net.NodeAddr, error) {

	nddb, err := s.DBConn.DB().GetNodesObject()

	if err != nil {
		return nil, err
	}

	nodes := []net.NodeAddr{}

	nddb.ForEach(func(k, v []byte) error {
		addr := string(v)
		node := net.NodeAddr{}
		node.LoadFromString(addr)

		nodes = append(nodes, node)
		return nil
	})

	return nodes, nil
}
func (s NodesListStorage) AddNodeToKnown(addr net.NodeAddr) {
	if !s.DBConn.CheckConnectionIsOpen() {
		// if connection is not opened when this function is called, we have to close it
		// we do this because this structre can be shared between threads.
		defer s.DBConn.CloseConnection()
	}
	//s.DBConn.Logger.Trace.Printf("AddNodeToKnown %s", addr.NodeAddrToString())

	nddb, err := s.DBConn.DB().GetNodesObject()

	if err != nil {
		s.DBConn.Logger.Trace.Printf("err %s", err.Error())
		return
	}
	address := addr.NodeAddrToString()
	key := []byte(address)

	nddb.PutNode(key, key)

	return
}
func (s NodesListStorage) RemoveNodeFromKnown(addr net.NodeAddr) {
	if !s.DBConn.CheckConnectionIsOpen() {
		// if connection is not opened when this function is called, we have to close it
		// we do this because this structre can be shared between threads.
		defer s.DBConn.CloseConnection()
	}
	nddb, err := s.DBConn.DB().GetNodesObject()

	if err != nil {
		return
	}
	address := addr.NodeAddrToString()
	key := []byte(address)
	nddb.DeleteNode(key)
	return
}
func (s NodesListStorage) GetCountOfKnownNodes() (int, error) {

	nddb, err := s.DBConn.DB().GetNodesObject()

	if err != nil {
		return 0, err
	}

	return nddb.GetCount()
}
