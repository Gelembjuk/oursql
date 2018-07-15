package consensus

import (
	"crypto/ecdsa"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
)

const (
	BlockPrepare_Error                              = 0
	BlockPrepare_Done                               = 1
	BlockPrepare_NoTransactions                     = 2
	BlockPrepare_NotGoodTime                        = 3
	SQLProcessingResultError                        = 0 // error
	SQLProcessingResultExecuted                     = 1 // Query doesn't need signature . It was executed .
	SQLProcessingResultPubKeyRequired               = 2 // Query needs signature and all other info. Data to sign is not yet preared (pubkey was not provided)
	SQLProcessingResultSignatureRequired            = 3 // Query needs signature. TX was prepared and data to sign is retuned
	SQLProcessingResultTranactionComplete           = 4 // Query needs signature. TX was created with provied signature
	SQLProcessingResultTranactionCompleteInternally = 5 // Query needs signature. TX was created with internal keys and completed
)

type BlockMakerInterface interface {
	SetDBManager(DB database.DBManager)
	SetLogManager(Logger *utils.LoggerMan)
	SetMinterAddress(minter string)
	PrepareNewBlock() (int, error)
	SetPreparedBlock(block *structures.Block) error
	IsBlockPrepared() bool
	CompleteBlock() (*structures.Block, error)
	VerifyBlock(block *structures.Block) error
}

type SQLTransactionsInterface interface {
	NewQuery(sql string, pubKey []byte) (uint, []byte, []byte, *structures.Transaction, error)
	NewQuerySigned(txEncoded []byte, signature []byte) (*structures.Transaction, error)
	NewQueryByNode(sql string, pubKey []byte, privKey ecdsa.PrivateKey) (uint, *structures.Transaction, error)
	NewQueryFromProxy(sql string) (*structures.Transaction, error)
	ExecuteOnBlockAdd(txlist []structures.Transaction) error
	ExecuteOnBlockCancel(txlist []structures.Transaction) error
}

func NewBlockMakerManager(minter string, DB database.DBManager, Logger *utils.LoggerMan) (BlockMakerInterface, error) {
	bm := &NodeBlockMaker{}
	bm.DB = DB
	bm.Logger = Logger
	bm.MinterAddress = minter
	return bm, nil
}

func NewSQLQueryManager(DB database.DBManager, Logger *utils.LoggerMan, pubKey []byte, privKey ecdsa.PrivateKey) (SQLTransactionsInterface, error) {
	qm := &queryManager{}
	qm.DB = DB
	qm.Logger = Logger
	qm.pubKey = pubKey
	qm.privKey = privKey

	return qm, nil
}
