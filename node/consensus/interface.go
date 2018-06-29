package consensus

import (
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
)

const BlockPrepare_Error = 0
const BlockPrepare_Done = 1
const BlockPrepare_NoTransactions = 2
const BlockPrepare_NotGoodTime = 3

type ConsensusInterface interface {
	SetDBManager(DB database.DBManager)
	SetLogManager(Logger *utils.LoggerMan)
	SetMinterAddress(minter string)
	PrepareNewBlock() (int, error)
	SetPreparedBlock(block *structures.Block) error
	IsBlockPrepared() bool
	CompleteBlock() (*structures.Block, error)
	VerifyBlock(block *structures.Block) error
}

func NewConsensusManager(minter string, DB database.DBManager, Logger *utils.LoggerMan) (ConsensusInterface, error) {
	bm := &NodeBlockMaker{}
	bm.DB = DB
	bm.Logger = Logger
	bm.MinterAddress = minter
	return bm, nil
}
