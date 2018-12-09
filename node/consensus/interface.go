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
	SQLProcessingResultCanBeExecuted                = 6 // Query doesn't need signature . It was NOT executed. A proxy can pass it to a server
)

// The structure to return information on new query request from proxy
// This includes a status , data to sign (if needed), new transaction (if was created)
// The structure can include error, so no need to have error response separately
type QueryFromProxyResult struct {
	Status       uint8
	TX           *structures.Transaction
	TXData       []byte
	StringToSign []byte
	ReplaceQuery string
	ErrorCode    uint16
	Error        error
}

type BlockMakerInterface interface {
	SetDBManager(DB database.DBManager)
	SetLogManager(Logger *utils.LoggerMan)
	SetMinterAddress(minter string)
	PrepareNewBlock() (int, error)
	SetPreparedBlock(block *structures.Block) error
	IsBlockPrepared() bool
	GetPreparedBlockTransactionsIDs() ([][]byte, error) // returns list of transactions in prepared block
	CompleteBlock() (*structures.Block, error)
	VerifyBlock(block *structures.Block, flags int) error
	AddTransactionToPool(tx *structures.Transaction, flags int) error
}

type SQLTransactionsInterface interface {
	NewQuery(sql string, pubKey []byte) (uint, []byte, []byte, *structures.Transaction, error)
	NewQuerySigned(txEncoded []byte, signature []byte) (*structures.Transaction, error)
	NewQueryByNode(sql string, pubKey []byte, privKey ecdsa.PrivateKey) (uint, *structures.Transaction, error)
	NewQueryByNodeInit(sql string, pubKey []byte, privKey ecdsa.PrivateKey) (tx *structures.Transaction, err error)
	NewQueryFromProxy(sql string) QueryFromProxyResult
	RepeatTransactionsFromCanceledBlocks(txList []structures.Transaction) error
}

func NewBlockMakerManager(config *ConsensusConfig, minter string, DB database.DBManager, Logger *utils.LoggerMan) BlockMakerInterface {
	bm := &NodeBlockMaker{}
	bm.DB = DB
	bm.Logger = Logger
	bm.MinterAddress = minter
	bm.config = config
	return bm
}

func NewSQLQueryManager(config *ConsensusConfig, DB database.DBManager, Logger *utils.LoggerMan, pubKey []byte, privKey ecdsa.PrivateKey) (SQLTransactionsInterface, error) {
	qm := &queryManager{}
	qm.DB = DB
	qm.Logger = Logger
	qm.pubKey = pubKey
	qm.privKey = privKey
	qm.config = config

	return qm, nil
}
