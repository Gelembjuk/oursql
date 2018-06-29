package database

const transactionsTable = "transactions"
const transactionsOutputsTable = "transactionsoutputs"

type Tranactions struct {
	DB                       *MySQLDB
	transactionsTable        string
	transactionsOutputsTable string
}

func (txs *Tranactions) getTransactionsTable() string {
	if txs.transactionsTable == "" {
		txs.transactionsTable = txs.DB.tablesPrefix + transactionsTable
	}
	return txs.transactionsTable
}

func (txs *Tranactions) getTransactionsOutputsTable() string {
	if txs.transactionsOutputsTable == "" {
		txs.transactionsOutputsTable = txs.DB.tablesPrefix + transactionsOutputsTable
	}
	return txs.transactionsOutputsTable
}

// Init database
func (txs *Tranactions) InitDB() error {
	err := txs.DB.CreateTable(txs.getTransactionsTable(), "VARBINARY(100)", "BLOB")

	if err != nil {
		return err
	}
	return txs.DB.CreateTable(txs.getTransactionsOutputsTable(), "VARBINARY(100)", "BLOB")
}

// transacet tables
func (txs *Tranactions) TruncateDB() error {
	err := txs.DB.Truncate(txs.getTransactionsTable())

	if err != nil {
		return err
	}
	return txs.DB.Truncate(txs.getTransactionsOutputsTable())
}

// Save link between TX and block hash
func (txs *Tranactions) PutTXToBlockLink(txID []byte, blockHash []byte) error {
	return txs.DB.Put(txs.getTransactionsTable(), txID, blockHash)
}

// Get block hash for TX
func (txs *Tranactions) GetBlockHashForTX(txID []byte) ([]byte, error) {
	return txs.DB.Get(txs.getTransactionsTable(), txID)
}

// Delete link between TX and a block hash
func (txs *Tranactions) DeleteTXToBlockLink(txID []byte) error {
	return txs.DB.Delete(txs.getTransactionsTable(), txID)
}

// Save spent outputs for TX
func (txs *Tranactions) PutTXSpentOutputs(txID []byte, outputs []byte) error {
	return txs.DB.Put(txs.getTransactionsOutputsTable(), txID, outputs)
}

// Get spent outputs for TX , seialised to bytes
func (txs *Tranactions) GetTXSpentOutputs(txID []byte) ([]byte, error) {
	return txs.DB.Get(txs.getTransactionsOutputsTable(), txID)
}

// Delete info about spent outputs for TX
func (txs *Tranactions) DeleteTXSpentData(txID []byte) error {
	return txs.DB.Delete(txs.getTransactionsOutputsTable(), txID)
}
