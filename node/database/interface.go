package database

import (
	"github.com/gelembjuk/oursql/lib/utils"
)

type DBManager interface {
	QM() DBQueryManager // get QueryManager object

	SetConfig(config DatabaseConfig) error
	SetLogger(logger *utils.LoggerMan) error
	GetLockerObject() DatabaseLocker
	SetLockerObject(lockerobj DatabaseLocker)

	InitDatabase() error
	CheckDBExists() (bool, error)

	CheckConnection() error
	OpenConnection() error
	CloseConnection() error
	IsConnectionOpen() bool

	GetBlockchainObject() (BlockchainInterface, error)
	GetTransactionsObject() (TranactionsInterface, error)
	GetUnapprovedTransactionsObject() (UnapprovedTransactionsInterface, error)
	GetUnspentOutputsObject() (UnspentOutputsInterface, error)
	GetNodesObject() (NodesInterface, error)
}

type DBQueryManager interface {
	Dump(file string) error
	Restore(file string) error
	Quote(value string) string
	ExecuteSQL(sql string) error
	ExecuteSQLExplain(sql string) (SQLExplainInfo, error)
	ExecuteSQLPrimaryKey(table string) (string, error)
	ExecuteSQLNextKeyValue(table string) (string, error)
	ExecuteSQLSelectRow(sqlcommand string) (data map[string]string, err error)
}

type SQLExplainInfo struct {
	Id           string
	SelectType   string
	Table        string
	Partitions   string
	Type         string
	PossibleKeys string
	Key          string
	KeyLen       int
	Ref          string
	Rows         int
	Filtered     string
	Extra        string
}

// locker interface. is empty for now. maybe in future we will have some methods
type DatabaseLocker interface {
}

type DatabaseConnection interface {
	Close() error
}
type ForEachKeyIteratorInterface func(key, value []byte) error

type BlockchainInterface interface {
	InitDB() error

	GetTopBlock() ([]byte, error)
	GetBlock(hash []byte) ([]byte, error)
	PutBlockOnTop(hash []byte, blockdata []byte) error
	PutBlock(hash []byte, blockdata []byte) error
	CheckBlockExists(hash []byte) (bool, error)
	DeleteBlock(hash []byte) error
	SaveTopHash(hash []byte) error
	GetTopHash() ([]byte, error)
	SaveFirstHash(hash []byte) error
	GetFirstHash() ([]byte, error)

	GetLocationInChain(hash []byte) (bool, []byte, []byte, error)
	BlockInChain(hash []byte) (bool, error)
	RemoveFromChain(hash []byte) error
	AddToChain(hash, prevHash []byte) error
}

type TranactionsInterface interface {
	InitDB() error
	TruncateDB() error
	PutTXToBlockLink(txID []byte, blockHash []byte) error
	GetBlockHashForTX(txID []byte) ([]byte, error)
	DeleteTXToBlockLink(txID []byte) error
	PutTXSpentOutputs(txID []byte, outputs []byte) error
	GetTXSpentOutputs(txID []byte) ([]byte, error)
	DeleteTXSpentData(txID []byte) error
}

type UnapprovedTransactionsInterface interface {
	InitDB() error
	TruncateDB() error
	ForEach(callback ForEachKeyIteratorInterface) error
	GetCount() (int, error)

	GetTransaction(txID []byte) ([]byte, error)
	PutTransaction(txID []byte, txdata []byte) error
	DeleteTransaction(txID []byte) error
}

type UnspentOutputsInterface interface {
	InitDB() error
	TruncateDB() error
	ForEach(callback ForEachKeyIteratorInterface) error
	GetCount() (int, error)

	GetDataForTransaction(txID []byte) ([]byte, error)
	DeleteDataForTransaction(txID []byte) error
	PutDataForTransaction(txID []byte, txData []byte) error
}

type NodesInterface interface {
	InitDB() error
	ForEach(callback ForEachKeyIteratorInterface) error
	GetCount() (int, error)

	PutNode(nodeID []byte, nodeData []byte) error
	DeleteNode(nodeID []byte) error
}
