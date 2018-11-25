'''
Test for 2 nodes and mixed operations - proxies + currency operations
'''
import _lib
import _transfers
import _sql
import _blocks
import _complex
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
    
    _lib.StartTestGroup("SQL Sync with Proxy")

    _lib.CleanTestFolders()
    
    datadir = _lib.CreateTestFolder('_1_')

    startnode.StartNodeWithoutBlockchain(datadir)
    
    address = startnode.InitBockchain(datadir)
    _complex.AddProxyToConfig(datadir, "localhost:40041")
    _complex.AddInternalKeyToConfig(datadir, address) # init internal signing
    
    startnode.StartNode(datadir, address, '30000')
    
    _lib.StartTestGroup("Do transactions")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    _sql.ExecuteSQLOnProxy(datadir,"CREATE TABLE test (a INT auto_increment PRIMARY KEY, b VARCHAR(20))")
    
    # check new table exists
    tables = _lib.DBGetRows(datadir,"SHOW TABLES",True)
    found = False
    for table in tables:
        if table[0] == "test":
            found = True
            break
    
    _lib.FatalAssert(found, "Table not found in the DB")
    
    blocks = _blocks.WaitBlocks(datadir, 2)
    time.sleep(2)
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row1'")
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET a=2,b='row2'")
    time.sleep(1)
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test (b) VALUES ('row3')")
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test",True)
    
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table")
    
    blocks = _blocks.WaitBlocks(datadir, 3)
    
    time.sleep(2)# while all caches are cleaned
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
    
    # update data
    _sql.ExecuteSQLOnProxy(datadir," update test SET b=\"row3 updated\" where a=3")
    _sql.ExecuteSQLOnProxy(datadir," update test SET b=\"row2 updated\" where a = '2'")
    
    blocks = _blocks.WaitBlocks(datadir, 4)
    time.sleep(1)# while all caches are cleaned
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test",True)
    for row in rows:
        if row[0]=="1":
            _lib.FatalAssert(row[1] == "row1","Row 1 value is wrong. Got: "+row[1])
            
        if row[0]=="2":
            _lib.FatalAssert(row[1] == "row2 updated","Row 2 value is wrong. Got: "+row[1])
            
        if row[0]=="3":
            _lib.FatalAssert(row[1] == "row3 updated","Row 3 value is wrong. Got: "+row[1])
    
    _sql.ExecuteSQLOnProxyFail(datadir,"INSERT INTO test SET a=2,b='row2'")

    _sql.ExecuteSQLOnProxy(datadir," DELETE  from   test where a=3")
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 2, "Must be 2 rows in a table")
    
    datadir2 = _lib.CreateTestFolder('_2_')
    
    address2 = initblockchain.ImportBockchain(datadir2,"localhost",'30000')
    
    _complex.AddProxyToConfig(datadir2, "localhost:40042")
    _complex.AddInternalKeyToConfig(datadir2, address2) # init internal signing
    
    startnode.StartNode(datadir2, address2,'30001', "Server 2")
    blocks = _blocks.WaitBlocks(datadir2, 4)
    
    # Send money to new node address
    txid1 = _transfers.Send(datadir,address,address2,1)
    
    time.sleep(2)
    
    # must be 2 delete transaction should be imported
    rows = _lib.DBGetRows(datadir2,"SELECT * FROM test",True)
    
    _lib.FatalAssert(len(rows) == 2, "Must be 2 rows in a table")
    
    _sql.ExecuteSQLOnProxy(datadir," DELETE  from   test where a=2")
    
    # send money again
    txid2 = _transfers.Send(datadir,address,address2,1)
    
    # should be 1 row on first node
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 1, "Must be 1 rows in a table")
    
    time.sleep(1)# give time to send transaction
    # and 2 row on second
    rows = _lib.DBGetRows(datadir2,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 1, "Must be 1 rows in a table")
    
    blocks = _blocks.WaitBlocks(datadir, 5)
    blocks = _blocks.WaitBlocks(datadir2, 5)
    time.sleep(1)
    
    # send money back
    
    txid3 = _transfers.Send(datadir2,address2,address,2)
    
    # insert on second node. check on first
    _sql.ExecuteSQLOnProxy(datadir2,"INSERT INTO test SET a=2,b='row2'")
    
    time.sleep(3)# give time to send transaction
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test",True)
    
    _lib.FatalAssert(len(rows) == 2, "Must be 2 rows in a table")
    
    # check balances
    bal1 = _transfers.GetBalance(datadir, address)
    bal2 = _transfers.GetBalance(datadir2, address2)
    
    _lib.FatalAssert(bal2[2] == -2, "Pending balance should be -2 for second address")
    _lib.FatalAssert(bal1[2] == 2, "Pending balance should be 2 for first address")
    
    startnode.StopNode(datadir)
    datadir = ""
    
    startnode.StopNode(datadir2)
    datadir2 = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    

    