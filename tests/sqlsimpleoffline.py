import _lib
import _transfers
import _sql
import _blocks
import re
import time
import startnode
import transactions
        
def test(testfilter):

    
    _lib.StartTestGroup("SQL basic")

    _lib.CleanTestFolders()
    datadir = _lib.CreateTestFolder()

    address = startnode.InitBockchain(datadir)
    
    _lib.StartTestGroup("Do transactions")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    tx1 = _sql.ExecuteSQL(datadir,address,"CREATE TABLE test (a INT auto_increment PRIMARY KEY, b VARCHAR(20))")
    
    # check new table exists
    tables = _lib.DBGetRows(datadir,"SHOW TABLES")
    found = False
    for table in tables:
        if table[0] == "test":
            found = True
            break
    
    _lib.FatalAssert(found, "Table not found in the DB")
    
    # now add data and make a block 
    _sql.ExecuteSQL(datadir,address,"INSERT INTO test SET b='row1'")
    _sql.ExecuteSQL(datadir,address,"INSERT INTO test SET b='row2', a=2")
    _sql.ExecuteSQL(datadir,address,"INSERT INTO test  (b) values ('row3')")
    _sql.ExecuteSQL(datadir,address,"INSERT INTO test  (a, b) values (4, 'row4')")
    _sql.ExecuteSQL(datadir,address,"INSERT INTO test (a, b) values (8, 'row5')")
    
    _sql.ExecuteSQL(datadir,address,"UPDATE test SET b='row1_u1' where a=1")
    _sql.ExecuteSQL(datadir,address,"UPDATE test SET b='row2_u1' where a = '2'")
    
    _sql.ExecuteSQL(datadir,address,"UPDATE test SET b='row1_u2' where a= 1")
    
    _sql.ExecuteSQL(datadir,address,"delete from test where a=3")
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test")
    
    _lib.FatalAssert(len(rows) == 4, "Must be 4 rows in a table")
    
    blockchash = _blocks.MintBlock(datadir,address)
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    # test execution when transactions are already in a block
    _sql.ExecuteSQLFailure(datadir,address,"INSERT INTO test SET b='row2', a=2")
    _sql.ExecuteSQLFailure(datadir,address,"delete from test where a=3")
    _sql.ExecuteSQLFailure(datadir,address,"UPDATE test SET b='upd' where a=  3")
    _sql.ExecuteSQLFailure(datadir,address,"UPDATE test SET b='upd2', a=3 where a=  2")# we don't allow to change key value
    
    _sql.ExecuteSQL(datadir,address,"INSERT INTO test (a, b) values (6, 'row6')")
    _sql.ExecuteSQL(datadir,address,"UPDATE test SET b='row8_u1' where a=8")
    _sql.ExecuteSQL(datadir,address,"delete from test where a=2")
    _sql.ExecuteSQL(datadir,address,"UPDATE test SET b='row8_u2' where a=8")
    _sql.ExecuteSQL(datadir,address,"UPDATE test SET b='row6_u1' where a=6")
    
    
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    

    