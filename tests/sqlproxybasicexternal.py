'''
Same as sqlproxybasic, but uses internal signing of transactions
'''
import _lib
import _transfers
import _complex
import _sql
import _blocks
import re
import time
import startnode
import transactions

datadir = ""

def aftertest(testfilter):
    global datadir
    
    if datadir != "":
        startnode.StopNode(datadir)
        
def test(testfilter):
    global datadir
    
    _lib.StartTestGroup("SQL Proxy basic")

    _lib.CleanTestFolders()
    datadir = _lib.CreateTestFolder()

    pub_key, pri_key = _lib.MakeWallet()

    startnode.StartNodeWithoutBlockchain(datadir)
    
    address = startnode.InitBockchain(datadir)
    _complex.AddProxyToConfig(datadir, "localhost:40041")
    
    startnode.StartNode(datadir, address, '30000')
    
    _lib.StartTestGroup("Do transactions")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    _sql.ExecuteSQLOnProxySign(datadir, "CREATE TABLE test (a INT auto_increment PRIMARY KEY, b VARCHAR(20))", pub_key, pri_key)
    
    # check new table exists
    tables = _lib.DBGetRows(datadir,"SHOW TABLES")
    found = False
    for table in tables:
        if table[0] == "test":
            found = True
            break
    
    _lib.FatalAssert(found, "Table not found in the DB")
    
    blocks = _blocks.WaitBlocks(datadir, 2)
    time.sleep(1)
    
    _sql.ExecuteSQLOnProxySign(datadir,"INSERT INTO test SET b='row1'", pub_key, pri_key)
    _sql.ExecuteSQLOnProxySign(datadir,"INSERT INTO test SET a=2,b='row2'", pub_key, pri_key)
    time.sleep(1)
    _sql.ExecuteSQLOnProxySign(datadir,"INSERT INTO test (b) VALUES ('row3')", pub_key, pri_key)
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test")
    
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table")
    
    blocks = _blocks.WaitBlocks(datadir, 3)
    
    time.sleep(1)# while all caches are cleaned
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
    
    for row in rows:
        if row[0] == "1":
            _lib.FatalAssert(row[1] == "row1", "Wrong value for row1")
        if row[0] == "2":
            _lib.FatalAssert(row[1] == "row2", "Wrong value for row2")
        if row[0] == "3":
            _lib.FatalAssert(row[1] == "row3", "Wrong value for row3")
    
    # update data
    _sql.ExecuteSQLOnProxySign(datadir," update test SET b=\"row3 updated\" where a=3", pub_key, pri_key)
    _sql.ExecuteSQLOnProxySign(datadir," update test SET b=\"row2 updated\" where a = '2'", pub_key, pri_key)
    
    blocks = _blocks.WaitBlocks(datadir, 4)
    time.sleep(1)# while all caches are cleaned
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test", True)
    for row in rows:
        if row[0]=="1":
            _lib.FatalAssert(row[1] == "row1","Row 1 value is wrong. Got: "+row[1])
            
        if row[0]=="2":
            _lib.FatalAssert(row[1] == "row2 updated","Row 2 value is wrong. Got: "+row[1])
            
        if row[0]=="3":
            _lib.FatalAssert(row[1] == "row3 updated","Row 3 value is wrong. Got: "+row[1])
    
    _sql.ExecuteSQLOnProxySignFail(datadir,"INSERT INTO test SET a=2,b='row2'", pub_key, pri_key)

    _sql.ExecuteSQLOnProxySign(datadir," DELETE  from   test where a=3", pub_key, pri_key)
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test", True)
    
    _lib.FatalAssert(len(rows) == 2, "Must be 2 rows in a table")
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    #cancel transaction. rollback should affect
    transactions.CancelTransaction(datadir,txlist.keys()[0]);
    
    # should be 0 unapproved transactions
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    # should be 3 rows again
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test", True)
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table")
    
    startnode.StopNode(datadir)
    datadir = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    

    