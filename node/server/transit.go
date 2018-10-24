package server

import (
	"errors"

	"github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/lib/utils"
)

/*
* This structure is used to keep data in memory
 */
type nodeTransit struct {
	Blocks        map[string][][]byte
	MaxKnownHeigh int
	Logger        *utils.LoggerMan
}

func (t *nodeTransit) Init(l *utils.LoggerMan) error {
	t.Logger = l
	t.Blocks = make(map[string][][]byte)

	return nil
}
func (t *nodeTransit) AddBlocks(fromaddr net.NodeAddr, blocks [][]byte) error {
	key := fromaddr.NodeAddrToString()

	_, ok := t.Blocks[key]

	if !ok {
		t.Blocks[key] = blocks
	} else {
		t.Blocks[key] = append(t.Blocks[key], blocks...)
	}

	return nil
}

func (t *nodeTransit) CleanBlocks(fromaddr net.NodeAddr) {
	key := fromaddr.NodeAddrToString()

	if _, ok := t.Blocks[key]; ok {
		delete(t.Blocks, key)
	}
}

func (t *nodeTransit) GetBlocksCount(fromaddr net.NodeAddr) int {
	if _, ok := t.Blocks[fromaddr.NodeAddrToString()]; ok {
		return len(t.Blocks[fromaddr.NodeAddrToString()])
	}
	return 0
}

func (t *nodeTransit) ShiftNextBlock(fromaddr net.NodeAddr) ([]byte, error) {
	key := fromaddr.NodeAddrToString()

	if _, ok := t.Blocks[key]; ok {
		data := t.Blocks[key][0][:]
		t.Blocks[key] = t.Blocks[key][1:]

		if len(t.Blocks[key]) == 0 {
			delete(t.Blocks, key)
		}

		return data, nil
	}

	return nil, errors.New("The address is not in blocks transit")
}
