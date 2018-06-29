package database

const unapprovedTransactionsTable = "unapprovedtransactions"

type UnapprovedTransactions struct {
	DB        *MySQLDB
	tableName string
}

func (uts *UnapprovedTransactions) getTableName() string {
	if uts.tableName == "" {
		uts.tableName = uts.DB.tablesPrefix + unapprovedTransactionsTable
	}
	return uts.tableName
}

// Init DB. create table
func (uts *UnapprovedTransactions) InitDB() error {
	return uts.DB.CreateTable(uts.getTableName(), "VARBINARY(100)", "LONGBLOB")
}

// execute functon for each key/value in the bucket
func (uts *UnapprovedTransactions) ForEach(callback ForEachKeyIteratorInterface) error {
	return uts.DB.forEachInTable(uts.getTableName(), callback)
}

// get count of records in the table
func (uts *UnapprovedTransactions) GetCount() (int, error) {
	return uts.DB.getCountInTable(uts.getTableName())
}

func (uts *UnapprovedTransactions) TruncateDB() error {
	return uts.DB.Truncate(uts.getTableName())
}

// returns transaction by ID if it exists
func (uts *UnapprovedTransactions) GetTransaction(txID []byte) ([]byte, error) {
	return uts.DB.Get(uts.getTableName(), txID)
}

// Add transaction record
func (uts *UnapprovedTransactions) PutTransaction(txID []byte, txdata []byte) error {
	return uts.DB.Put(uts.getTableName(), txID, txdata)
}

// delete transation from DB
func (uts *UnapprovedTransactions) DeleteTransaction(txID []byte) error {
	return uts.DB.Delete(uts.getTableName(), txID)
}
