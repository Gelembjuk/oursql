package database

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"database/sql"

	"github.com/JamesStewy/go-mysqldump"
	"github.com/gelembjuk/oursql/lib/utils"
	_ "github.com/go-sql-driver/mysql"
)

const (
	ClassNameNodes                  = "nodes"
	ClassNameBlockchain             = "blockchain"
	ClassNameTransactions           = "transactions"
	ClassNameUnapprovedTransactions = "unapprovedtransactions"
	ClassNameUnspentOutputs         = "unspentoutputs"
)

type MySQLDBManager struct {
	Logger     *utils.LoggerMan
	Config     DatabaseConfig
	conn       *sql.DB
	openedConn bool
	SessID     string
}

func (bdm *MySQLDBManager) SetConfig(config DatabaseConfig) error {
	bdm.Config = config

	return nil
}
func (bdm *MySQLDBManager) SetLogger(logger *utils.LoggerMan) error {
	bdm.Logger = logger

	return nil
}

// try to set up a connection to DB. and close it then
func (bdm *MySQLDBManager) CheckConnection() error {
	conn, err := bdm.getConnection()

	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Query("SHOW TABLES")

	if err != nil {
		return err
	}

	return nil
}

// set status of connection to open
func (bdm *MySQLDBManager) OpenConnection() error {
	//bdm.Logger.Trace.Println("open connection for " + reason)
	if bdm.openedConn {
		return nil
	}
	// real connection will be done when first object is created
	bdm.openedConn = true

	bdm.conn = nil

	return nil
}
func (bdm *MySQLDBManager) CloseConnection() error {
	if !bdm.openedConn {
		return nil
	}

	if bdm.conn != nil {
		bdm.conn.Close()
		bdm.conn = nil
	}

	bdm.openedConn = false
	return nil
}

func (bdm *MySQLDBManager) IsConnectionOpen() bool {
	return bdm.openedConn
}

// create empty database. must create all
// creates tables for BC
func (bdm *MySQLDBManager) InitDatabase() error {

	bdm.OpenConnection()

	defer bdm.CloseConnection()

	bc, err := bdm.GetBlockchainObject()

	if err != nil {
		return err
	}

	err = bc.InitDB()

	if err != nil {
		return err
	}

	txs, err := bdm.GetTransactionsObject()

	if err != nil {
		return err
	}

	err = txs.InitDB()

	if err != nil {
		return err
	}

	utx, err := bdm.GetUnapprovedTransactionsObject()

	if err != nil {
		return err
	}

	err = utx.InitDB()

	if err != nil {
		return err
	}

	uos, err := bdm.GetUnspentOutputsObject()

	if err != nil {
		return err
	}

	err = uos.InitDB()

	if err != nil {
		return err
	}

	ns, err := bdm.GetNodesObject()

	if err != nil {
		return err
	}

	err = ns.InitDB()

	if err != nil {
		return err
	}

	return nil
}

// Check if database was already inited
func (bdm *MySQLDBManager) CheckDBExists() (bool, error) {
	bc, err := bdm.GetBlockchainObject()

	if err != nil {
		return false, nil
	}

	tophash, err := bc.GetTopHash()

	if err != nil {
		return false, nil
	}

	if len(tophash) > 0 {
		return true, nil
	}

	return false, nil
}

// returns BlockChain Database structure. does all init
func (bdm *MySQLDBManager) GetBlockchainObject() (BlockchainInterface, error) {
	conn, err := bdm.getConnection()

	if err != nil {
		return nil, err
	}

	bc := Blockchain{}
	bc.DB = &MySQLDB{conn, bdm.Config.TablesPrefix, bdm.Logger}

	return &bc, nil
}

// returns Transaction Index Database structure. does al init
func (bdm *MySQLDBManager) GetTransactionsObject() (TranactionsInterface, error) {
	conn, err := bdm.getConnection()

	if err != nil {
		return nil, err
	}

	txs := Tranactions{}
	txs.DB = &MySQLDB{conn, bdm.Config.TablesPrefix, bdm.Logger}

	return &txs, nil
}

// returns Unapproved Transaction Database structure. does al init
func (bdm *MySQLDBManager) GetUnapprovedTransactionsObject() (UnapprovedTransactionsInterface, error) {
	conn, err := bdm.getConnection()

	if err != nil {
		return nil, err
	}

	uos := UnapprovedTransactions{}
	uos.DB = &MySQLDB{conn, bdm.Config.TablesPrefix, bdm.Logger}

	return &uos, nil
}

// returns Unspent Transactions Database structure. does al init
func (bdm *MySQLDBManager) GetUnspentOutputsObject() (UnspentOutputsInterface, error) {
	conn, err := bdm.getConnection()

	if err != nil {
		return nil, err
	}

	uts := UnspentOutputs{}
	uts.DB = &MySQLDB{conn, bdm.Config.TablesPrefix, bdm.Logger}

	return &uts, nil
}

// returns Nodes Database structure. does al init
func (bdm *MySQLDBManager) GetNodesObject() (NodesInterface, error) {
	conn, err := bdm.getConnection()

	if err != nil {
		return nil, err
	}

	ns := Nodes{}
	ns.DB = &MySQLDB{conn, bdm.Config.TablesPrefix, bdm.Logger}

	return &ns, nil
}

// returns DB connection, creates it if needed .
func (bdm *MySQLDBManager) getConnection() (*sql.DB, error) {

	if !bdm.openedConn {
		return nil, errors.New("Connection was not inited")
	}

	if bdm.conn != nil {
		return bdm.conn, nil
	}

	db, err := sql.Open("mysql", bdm.Config.GetMySQLConnString())

	if err != nil {
		return nil, err
	}
	//db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)

	bdm.conn = db

	return db, nil
}

func (bdm *MySQLDBManager) GetLockerObject() DatabaseLocker {
	return nil
}
func (bdm *MySQLDBManager) SetLockerObject(lockerobj DatabaseLocker) {

}

func (bdm *MySQLDBManager) Dump(file string) error {
	conn, err := bdm.getConnection()

	if err != nil {
		return err
	}
	// Register database with mysqldump
	dumpDir, _ := filepath.Abs(filepath.Dir(file))
	dumpFilename := filepath.Base(file)

	if strings.HasSuffix(dumpFilename, ".sql") {
		dumpFilename = dumpFilename[:len(dumpFilename)-4]
	}
	fmt.Printf("file name %s", dumpFilename)
	dumper, err := mysqldump.Register(conn, dumpDir, dumpFilename)

	if err != nil {
		return err
	}

	// Dump database to file
	_, err = dumper.Dump()

	if err != nil {
		return err
	}

	dumper.Close()
	return nil
}
func (bdm *MySQLDBManager) Restore(file string) error {
	connstr := bdm.Config.GetMySQLConnString() + "?multiStatements=true"
	db, err := sql.Open("mysql", connstr)

	if err != nil {
		return err
	}

	// load file to string
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	sql := string(b)

	_, err = db.Exec(sql)

	return err
}
func (bdm MySQLDBManager) ExecuteSQL(sql string) error {
	db, err := bdm.getConnection()

	if err != nil {
		return err
	}
	_, err = db.Exec(sql)
	return err
}
