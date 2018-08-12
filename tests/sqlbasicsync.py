import _lib
import _transfers
import _sql
import _blocks
import re
import time
import startnode
import transactions
import initblockchain

datadir = ""
datadir2 = ""

def aftertest(testfilter):
    global datadir,datadir2
    
    if datadir != "":
        startnode.StopNode(datadir)
    if datadir2 != "":
        startnode.StopNode(datadir2)
        
def test(testfilter):
    global datadir,datadir2
    
    _lib.StartTestGroup("SQL basic")

    _lib.CleanTestFolders()
    
    datadir = _lib.CreateTestFolder('_1_')
    datadir2 = _lib.CreateTestFolder('_2_')

    startnode.StartNodeWithoutBlockchain(datadir)
    address = startnode.InitBockchain(datadir)
    startnode.StartNode(datadir, address, '30000')
    
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
    
    blocks = _blocks.WaitBlocks(datadir, 2)
    
    tx2 = _sql.ExecuteSQL(datadir,address,"INSERT INTO test SET b='row1'")
    tx3 = _sql.ExecuteSQL(datadir,address,"INSERT INTO test SET a=2,b='row2'")
    time.sleep(1)
    tx4 = _sql.ExecuteSQL(datadir,address,"INSERT INTO test (b) VALUES ('row3')")
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test")
    
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table")
    
    blocks = _blocks.WaitBlocks(datadir, 3)
    
    time.sleep(1)# while all caches are cleaned
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
    
    # update data
    _sql.ExecuteSQL(datadir,address," update test SET b=\"row3 updated\" where a=3")
    _sql.ExecuteSQL(datadir,address," update test SET b=\"row2 updated\" where a = '2'")
    
    blocks = _blocks.WaitBlocks(datadir, 4)
    time.sleep(1)# while all caches are cleaned
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test")
    for row in rows:
        if row[0]=="1":
            _lib.FatalAssert(row[1] == "row1","Row 1 value is wrong. Got: "+row[1])
            
        if row[0]=="2":
            _lib.FatalAssert(row[1] == "row2 updated","Row 2 value is wrong. Got: "+row[1])
            
        if row[0]=="3":
            _lib.FatalAssert(row[1] == "row3 updated","Row 3 value is wrong. Got: "+row[1])
    
    error = _sql.ExecuteSQLFailure(datadir,address,"INSERT INTO test SET a=2,b='row2'")

    txid = _sql.ExecuteSQL(datadir,address," DELETE  from   test where a=3")
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test")
    _lib.FatalAssert(len(rows) == 2, "Must be 2 rows in a table")
    
    address2 = initblockchain.ImportBockchain(datadir2,"localhost",'30000')
    
    startnode.StartNode(datadir2, address2,'30001', "Server 2")
    blocks = _blocks.WaitBlocks(datadir2, 4)
    
    # must be 3 rows because delete transaction was not posted to that node
    rows = _lib.DBGetRows(datadir2,"SELECT * FROM test")
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table")
    
    txid = _sql.ExecuteSQL(datadir,address," DELETE  from   test where a=2")
    
    # should be 1 row on first node
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test")
    _lib.FatalAssert(len(rows) == 1, "Must be 1 rows in a table")
    
    time.sleep(1)# give time to send transaction
    # and 2 row on second
    rows = _lib.DBGetRows(datadir2,"SELECT * FROM test")
    _lib.FatalAssert(len(rows) == 2, "Must be 2 rows in a table")
    
    startnode.StopNode(datadir)
    datadir = ""
    
    startnode.StopNode(datadir2)
    datadir2 = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    

    