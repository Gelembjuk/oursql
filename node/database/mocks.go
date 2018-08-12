package database

import (
	"github.com/gelembjuk/oursql/lib/utils"
)

type mockMySQLDBManager struct {
	ER        *SQLExplainInfo
	KeyColumn string
}

func GetDBManagerMock() mockMySQLDBManager {
	return mockMySQLDBManager{}
}
func (bdm *mockMySQLDBManager) QM() DBQueryManager {
	return bdm
}
func (bdm mockMySQLDBManager) SetConfig(config DatabaseConfig) error {
	return nil
}
func (bdm mockMySQLDBManager) SetLogger(logger *utils.LoggerMan) error {
	return nil
}

func (bdm mockMySQLDBManager) CheckConnection() error {
	return nil
}
func (bdm mockMySQLDBManager) OpenConnection() error {
	return nil
}
func (bdm mockMySQLDBManager) CloseConnection() error {
	return nil
}
func (bdm mockMySQLDBManager) IsConnectionOpen() bool {
	return true
}
func (bdm mockMySQLDBManager) InitDatabase() error {
	return nil
}
func (bdm mockMySQLDBManager) CheckDBExists() (bool, error) {
	return false, nil
}
func (bdm mockMySQLDBManager) GetBlockchainObject() (BlockchainInterface, error) {
	bc := Blockchain{}
	return &bc, nil
}
func (bdm mockMySQLDBManager) GetTransactionsObject() (TranactionsInterface, error) {
	txs := Tranactions{}
	return &txs, nil
}
func (bdm mockMySQLDBManager) GetUnapprovedTransactionsObject() (UnapprovedTransactionsInterface, error) {
	uos := UnapprovedTransactions{}
	return &uos, nil
}
func (bdm mockMySQLDBManager) GetUnspentOutputsObject() (UnspentOutputsInterface, error) {
	uts := UnspentOutputs{}
	return &uts, nil
}
func (bdm mockMySQLDBManager) GetNodesObject() (NodesInterface, error) {
	ns := Nodes{}
	return &ns, nil
}
func (bdm mockMySQLDBManager) GetLockerObject() DatabaseLocker {
	return nil
}
func (bdm mockMySQLDBManager) SetLockerObject(lockerobj DatabaseLocker) {
}
func (bdm mockMySQLDBManager) Dump(file string) error {
	return nil
}
func (bdm mockMySQLDBManager) Restore(file string) error {
	return nil
}
func (bdm mockMySQLDBManager) ExecuteSQL(sql string) error {
	return nil
}

// set explain info to return when requested
func (bdm *mockMySQLDBManager) SetSQLExplain(si *SQLExplainInfo) {
	bdm.ER = si
}
func (bdm mockMySQLDBManager) ExecuteSQLExplain(sql string) (SQLExplainInfo, error) {
	if bdm.ER != nil {
		return *bdm.ER, nil
	}
	return SQLExplainInfo{}, nil
}
func (bdm mockMySQLDBManager) ExecuteSQLPrimaryKey(table string) (column string, err error) {

	return bdm.KeyColumn, nil
}

func (bdm mockMySQLDBManager) ExecuteSQLSelectRow(sqlcommand string) (data map[string]string, err error) {
	return
}

func (bdm mockMySQLDBManager) ExecuteSQLNextKeyValue(table string) (string, error) {
	return "", nil
}
