package database

import (
	"database/sql"
	"encoding/hex"
	"strconv"

	"github.com/gelembjuk/oursql/lib/utils"
)

type MySQLDB struct {
	db           *sql.DB
	tablesPrefix string
	Logger       *utils.LoggerMan
}

// close DB connection
func (bdb *MySQLDB) Close() error {
	if bdb.db == nil {
		return nil
	}
	bdb.db.Close()
	bdb.db = nil

	return nil
}

// execute callback function for each record in a table
func (bdb *MySQLDB) forEachInTable(table string, callback ForEachKeyIteratorInterface) error {
	var k string
	var v string

	offset := 0

	for {
		err := bdb.db.QueryRow("SELECT * FROM "+table+" ORDER BY v LIMIT "+strconv.Itoa(offset)+",1").Scan(&k, &v)

		switch {
		case err == sql.ErrNoRows:
			return nil
		case err != nil:
			return err
		}
		err = callback(bdb.decodeKey(k), bdb.decodeValue(v))

		if err, ok := err.(*DBError); ok {
			if err.IsKind(DBCursorBreak) {
				// the function wants to break the loop
				return nil
			}
		}

		if err != nil {
			return err
		}

		offset = offset + 1
	}

	return nil
}

// get number of rows in a table
func (bdb *MySQLDB) getCountInTable(table string) (int, error) {
	var c int
	err := bdb.db.QueryRow("SELECT count(*) as c FROM " + table).Scan(&c)

	switch {
	case err != nil:
		return 0, err
	default:
		return c, nil
	}
}

// Get record from DB
func (bdb *MySQLDB) Get(table string, k []byte) ([]byte, error) {
	var v string
	s := "SELECT v FROM " + table + " WHERE k='" + bdb.encodeKey(k) + "'"

	err := bdb.db.QueryRow(s).Scan(&v)

	switch {
	case err == sql.ErrNoRows:

		return nil, nil
	case err != nil:

		return nil, err
	default:

		return bdb.decodeValue(v), nil
	}
}

// Put record in DB
func (bdb *MySQLDB) Put(table string, k, v []byte) error {
	ve := bdb.encodeValue(v)
	_, err := bdb.db.Exec("INSERT INTO "+table+" VALUES ( ? , ? ) ON DUPLICATE KEY UPDATE v=?", bdb.encodeKey(k), ve, ve)
	return err
}

// Delete record from DB
func (bdb *MySQLDB) Delete(table string, k []byte) error {
	_, err := bdb.db.Exec("DELETE FROM "+table+" WHERE k= ? ", bdb.encodeKey(k))
	return err
}

// truncate table
func (bdb *MySQLDB) Truncate(table string) error {
	_, err := bdb.db.Exec("TRUNCATE TABLE " + table)
	return err
}

// create key value table
func (bdb *MySQLDB) CreateTable(table string, keytype string, valuetype string) error {
	_, err := bdb.db.Exec("CREATE TABLE " + table + " ( k " + keytype + " PRIMARY KEY, v " + valuetype + " )")
	return err
}

// encode bytes to string
func (bdb *MySQLDB) encodeKey(k []byte) string {
	return hex.EncodeToString(k)
}

// decode bytes from string
func (bdb *MySQLDB) decodeKey(k string) []byte {
	kb, _ := hex.DecodeString(k)
	return kb
}

// encode value
// TODO maybe it is better to have base64 here ?
func (bdb *MySQLDB) encodeValue(v []byte) string {
	return hex.EncodeToString(v)
}

// decode value from string
func (bdb *MySQLDB) decodeValue(v string) []byte {
	vb, _ := hex.DecodeString(v)
	return vb
}
