package nodemanager

import (
	"sync"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
)

type Database struct {
	db        database.DBManager
	Logger    *utils.LoggerMan
	Config    database.DatabaseConfig
	lockerObj database.DatabaseLocker
	locallock *sync.Mutex
}

func (db *Database) DB() database.DBManager {
	db.OpenConnectionIfNeeded("", "")

	return db.db
}

// do initial actions
func (db *Database) Init() {
	// init locking system. one object will be used
	// in all goroutines

	db.locallock = &sync.Mutex{}
	db.PrepareConnection("")
	db.lockerObj = db.db.GetLockerObject()
	db.CleanConnection()
}

// prepare database before the first user
func (db *Database) InitDatabase() error {
	db.PrepareConnection("")
	err := db.db.InitDatabase()
	db.CleanConnection()
	return err
}

// Clone database object. all is clonned except locker object.
// locker object is shared between all objects
func (db *Database) Clone() Database {
	return *db
	/*
		ndb := Database{}
		ndb.locallock = &sync.Mutex{}
		ndb.SetLogger(db.Logger)
		ndb.SetConfig(db.Config)
		ndb.lockerObj = db.lockerObj

		return ndb
	*/
}

func (db *Database) SetLogger(Logger *utils.LoggerMan) {
	db.Logger = Logger
}

func (db *Database) SetConfig(config database.DatabaseConfig) {
	db.Config = config
}

// check if connection to DB can be set
func (db *Database) CheckConnection() error {
	return db.DB().CheckConnection()
}

// open DB connection if it is not yet opened
func (db *Database) OpenConnection(sessid string) error {
	//db.Logger.Trace.Printf("OpenConn in DB man %s", reason)

	if db.db != nil {
		return nil
	}
	db.PrepareConnection(sessid)

	// this will prevent creation of this object from other go routine
	//db.locallock.Lock()

	return db.db.OpenConnection()
}

func (db *Database) PrepareConnection(sessid string) {
	obj := &database.MySQLDBManager{}
	obj.SessID = sessid
	db.db = obj
	db.db.SetLogger(db.Logger)
	db.db.SetConfig(db.Config)

	if db.lockerObj != nil {
		db.db.SetLockerObject(db.lockerObj)
	}
}

func (db *Database) CloseConnection() error {
	//db.Logger.Trace.Printf("CloseConnection")
	// we don't close connection. this is controled inside SQL package
	return nil
	/*
		if db.db == nil {
			return nil
		}
		// now allow other go routine to create connection using same object
		//db.locallock.Unlock()
		db.db.CloseConnection()

		db.CleanConnection()

		return nil
	*/
}

// clean connection object
func (db *Database) CleanConnection() {
	db.db = nil
}

// open connection if it is not yet opened
func (db *Database) OpenConnectionIfNeeded(reason string, sessid string) bool {
	if db.db != nil {
		return false
	}

	err := db.OpenConnection(sessid)

	if err != nil {
		return false
	}

	return true
}

// check if connection was opened
func (db *Database) CheckConnectionIsOpen() bool {
	if db.db != nil {
		return true
	}
	return false
}

// dump DB to file
func (db *Database) Dump(file string) error {
	return db.DB().QM().Dump(file)
}

// restore from dump
func (db *Database) Restore(file string) error {
	return db.DB().QM().Restore(file)
}
