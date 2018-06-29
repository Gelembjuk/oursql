package database

const unspentTransactionsTable = "unspentoutputstransactions"

type UnspentOutputs struct {
	DB        *MySQLDB
	tableName string
}

// Get table name
func (uos *UnspentOutputs) getTableName() string {
	if uos.tableName == "" {
		uos.tableName = uos.DB.tablesPrefix + unspentTransactionsTable
	}
	return uos.tableName
}

// Init DB. create table
func (uos *UnspentOutputs) InitDB() error {
	return uos.DB.CreateTable(uos.getTableName(), "VARBINARY(100)", "LONGBLOB")
}

// execute functon for each key/value in the bucket
func (uos *UnspentOutputs) ForEach(callback ForEachKeyIteratorInterface) error {
	return uos.DB.forEachInTable(uos.getTableName(), callback)
}

// get count of records in the table
func (uos *UnspentOutputs) GetCount() (int, error) {
	return uos.DB.getCountInTable(uos.getTableName())
}

func (uos *UnspentOutputs) TruncateDB() error {
	return uos.DB.Truncate(uos.getTableName())
}

func (uos *UnspentOutputs) GetDataForTransaction(txID []byte) ([]byte, error) {
	return uos.DB.Get(uos.getTableName(), txID)
}

func (uos *UnspentOutputs) DeleteDataForTransaction(txID []byte) error {
	return uos.DB.Delete(uos.getTableName(), txID)
}
func (uos *UnspentOutputs) PutDataForTransaction(txID []byte, txData []byte) error {
	return uos.DB.Put(uos.getTableName(), txID, txData)
}
