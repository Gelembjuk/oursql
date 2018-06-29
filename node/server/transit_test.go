package server

import (
	"testing"

	"github.com/gelembjuk/oursql/lib"
)

func TestAddBlockSimple(t *testing.T) {
	tr := nodeTransit{}
	tr.Init(nil)

	addr := lib.NodeAddr{"localhost", 20000}

	blocks := [][]byte{{1, 2, 4}, {4, 5, 6}}

	tr.AddBlocks(addr, blocks)

	if tr.GetBlocksCount(addr) != 2 {
		t.Fatalf("Expected 2 blocks")
	}

	if tr.GetBlocksCount(lib.NodeAddr{}) != 0 {
		t.Fatalf("Expected 0 blocks")
	}
}
