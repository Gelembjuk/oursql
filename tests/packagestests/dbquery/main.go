package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/config"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/dbquery"
)

func main() {
	logger := utils.CreateLoggerStdout()
	dbconf := getDBConfig()

	dbconn := &database.MySQLDBManager{}

	dbconn.SetLogger(logger)
	dbconn.SetConfig(dbconf)

	err := dbconn.OpenConnection()

	if err != nil {
		log.Fatalln(err.Error())
	}

	cleanBeforeTest(dbconn)

	q := dbquery.NewQueryProcessor(dbconn, logger)

	createTestTables(dbconn, q)

	insertTest(dbconn, q)

	dropTestTables(dbconn, q)

	fmt.Println("Done!")

}
func cleanBeforeTest(dbconn *database.MySQLDBManager) {
	// drop table if exists to start testing
	err := dbconn.ExecuteSQL("DROP TABLE IF EXISTS test")

	if err != nil {
		log.Fatalln(err.Error())
	}

	err = dbconn.ExecuteSQL("DROP TABLE IF EXISTS testnai")

	if err != nil {
		log.Fatalln(err.Error())
	}
}
func createTestTables(dbconn *database.MySQLDBManager, q dbquery.QueryProcessorInterface) {
	qparsed, err := q.ParseQuery("CREATE TABLE test (a HZ primary key auto_increment, b varchar(50))")

	if err != nil {
		log.Fatalln(err.Error())
	}

	_, err = q.ExecuteParsedQuery(qparsed)

	if err == nil {
		log.Fatalln("Should be error on wrong create SQL")
	}

	qparsed, err = q.ParseQuery("CREATE TABLE test (a int primary key auto_increment, b varchar(50))")

	if err != nil {
		log.Fatalln(err.Error())
	}

	sqlupdate, err := q.ExecuteParsedQuery(qparsed)

	if err != nil {
		log.Fatalln(err.Error())
	}

	if string(sqlupdate.ReferenceID) != "test:*" {
		log.Fatalln("Unexpected RefID " + string(sqlupdate.ReferenceID))
	}
	// select from this table. should not be error
	err = dbconn.ExecuteSQL("SELECT * FROM test")

	if err != nil {
		log.Fatalln(err.Error())
	}

	// get primary key to check this function works
	keyCol, err := dbconn.QM().ExecuteSQLPrimaryKey(qparsed.Structure.GetTable())

	if err != nil {
		log.Fatalln(err.Error())
	}

	if keyCol != "a" {
		log.Fatalln("Wrong key column for table test")
	}

	qparsed, err = q.ParseQuery("CREATE TABLE testnai (a int primary key , b varchar(50))")

	if err != nil {
		log.Fatalln(err.Error())
	}

	sqlupdate, err = q.ExecuteParsedQuery(qparsed)

	if err != nil {
		log.Fatalln(err.Error())
	}

	if string(sqlupdate.ReferenceID) != "testnai:*" {
		log.Fatalln("Unexpected RefID " + string(sqlupdate.ReferenceID))
	}
	// select from this table. should not be error
	err = dbconn.ExecuteSQL("SELECT * FROM testnai")

	if err != nil {
		log.Fatalln(err.Error())
	}

	// get primary key to check this function works
	keyCol, err = dbconn.QM().ExecuteSQLPrimaryKey(qparsed.Structure.GetTable())

	if err != nil {
		log.Fatalln(err.Error())
	}

	if keyCol != "a" {
		log.Fatalln("Wrong key column for table test")
	}

	// get primary key to check this function works
	keyCol, err = dbconn.QM().ExecuteSQLPrimaryKey("notexistenttable")

	if err == nil {
		log.Fatalln("should be error if table does not exist")
	}
}
func dropTestTables(dbconn *database.MySQLDBManager, q dbquery.QueryProcessorInterface) {
	sqlupdate, err := q.ExecuteQuery("DROP TABLE test")

	if err != nil {
		log.Fatalln(err.Error())
	}

	if string(sqlupdate.ReferenceID) != "test:*" {
		log.Fatalln("Unexpected RefID " + string(sqlupdate.ReferenceID))
	}
	sqlupdate, err = q.ExecuteQuery("DROP TABLE testnai")

	if err != nil {
		log.Fatalln(err.Error())
	}

	if string(sqlupdate.ReferenceID) != "testnai:*" {
		log.Fatalln("Unexpected RefID " + string(sqlupdate.ReferenceID))
	}
}
func insertTest(dbconn *database.MySQLDBManager, q dbquery.QueryProcessorInterface) {
	sql := " INSERT into test set b='x'"
	sqlupdate, err := q.ExecuteQuery(sql)

	if err != nil {
		log.Fatalln(err.Error())
	}

	if string(sqlupdate.ReferenceID) != "test:1" {
		log.Fatalln("Unexpected RefID " + string(sqlupdate.ReferenceID))
	}

	if string(sqlupdate.RollbackQuery) != "DELETE FROM test WHERE a='1'" {
		log.Fatalln("Unexpected Rolback Query: " + string(sqlupdate.RollbackQuery) + " for: " + sql)
	}

	sql = " INSERT into test set b='x', a =2"
	sqlupdate, err = q.ExecuteQuery(sql)

	if err != nil {
		log.Fatalln(err.Error())
	}
	if string(sqlupdate.ReferenceID) != "test:2" {
		log.Fatalln("Unexpected RefID " + string(sqlupdate.ReferenceID))
	}
	if string(sqlupdate.RollbackQuery) != "DELETE FROM test WHERE a='2'" {
		log.Fatalln("Unexpected Rolback Query: " + string(sqlupdate.RollbackQuery) + " for: " + sql)
	}

	sql = " INSERT into testnai set b='x', a =2"
	sqlupdate, err = q.ExecuteQuery(sql)

	if err != nil {
		log.Fatalln(err.Error())
	}
	if string(sqlupdate.ReferenceID) != "testnai:2" {
		log.Fatalln("Unexpected RefID " + string(sqlupdate.ReferenceID))
	}
	if string(sqlupdate.RollbackQuery) != "DELETE FROM testnai WHERE a='2'" {
		log.Fatalln("Unexpected Rolback Query: " + string(sqlupdate.RollbackQuery) + " for: " + sql)
	}

	sql = " INSERT into testnai (a ,b) values (3,'bbbbb')"
	sqlupdate, err = q.ExecuteQuery(sql)

	if err != nil {
		log.Fatalln(err.Error())
	}
	if string(sqlupdate.ReferenceID) != "testnai:3" {
		log.Fatalln("Unexpected RefID " + string(sqlupdate.ReferenceID))
	}
	if string(sqlupdate.RollbackQuery) != "DELETE FROM testnai WHERE a='3'" {
		log.Fatalln("Unexpected Rolback Query: " + string(sqlupdate.RollbackQuery) + " for: " + sql)
	}
}
func updateTest(dbconn *database.MySQLDBManager, q dbquery.QueryProcessorInterface) {
	sql := " INSERT into test set b='x'"
	sqlupdate, err := q.ExecuteQuery(sql)

	if err != nil {
		log.Fatalln(err.Error())
	}

	if string(sqlupdate.ReferenceID) != "test:1" {
		log.Fatalln("Unexpected RefID " + string(sqlupdate.ReferenceID))
	}

	if string(sqlupdate.RollbackQuery) != "DELETE FROM test WHERE a='1'" {
		log.Fatalln("Unexpected Rolback Query: " + string(sqlupdate.RollbackQuery) + " for: " + sql)
	}

}
func getDBConfig() database.DatabaseConfig {
	// load from config file
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))

	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(0)
	}
	input, ierr := config.GetAppInputFromDir(dir)

	if ierr != nil {
		// something went wrong when parsing input data
		fmt.Printf("Error: %s\n", ierr.Error())
		os.Exit(0)
	}
	return input.Database
}
