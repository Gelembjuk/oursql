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
    
    _lib.StartTestGroup("SQL Sync with Proxy. care offline")

    _lib.CleanTestFolders()
    
    datadir = _lib.CreateTestFolder('_1_')
    
    startnode.StartNodeWithoutBlockchain(datadir)
    
    address = startnode.InitBockchain(datadir)
    _complex.AddProxyToConfig(datadir, "localhost:40041")
    _complex.AddInternalKeyToConfig(datadir, address) # init internal signing
    
    startnode.StartNode(datadir, address, '30000')
    
    _lib.StartTestGroup("Do initial transactions")

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
    time.sleep(1)
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row1'")
    
    datadir2 = _lib.CreateTestFolder('_2_')
    address2 = initblockchain.ImportBockchain(datadir2,"localhost",'30000')
    
    _complex.AddProxyToConfig(datadir2, "localhost:40042")
    _complex.AddInternalKeyToConfig(datadir2, address2) # init internal signing
    
    startnode.StartNode(datadir2, address2,'30001', "Server 2")
    blocks = _blocks.WaitBlocks(datadir2, 2)
    
    _sql.ExecuteSQLOnProxy(datadir2,"INSERT INTO test SET b='row1.2'")
    
    # must be 1 row , because the first row from the first node is not yet in a block and is not synced
    rows = _lib.DBGetRows(datadir2,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 1, "Must be 1 row in a table on node 2")
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row2'")
    # should be 1 row on first node
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 2, "Must be 2 rows in a table on node 1")
    
    # wait 3-rd block
    blocks = _blocks.WaitBlocks(datadir, 3)
    
    time.sleep(1)# give time to send transaction
    
    # wait 3-rd block on a second node
    blocks = _blocks.WaitBlocks(datadir2, 3)
    # and 2 row on second
    rows2 = _lib.DBGetRows(datadir2,"SELECT * FROM test ORDER BY a",True)
    _lib.FatalAssert(len(rows2) == 2, "Must be 2 rows in a table")
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test ORDER BY a",True)
    _lib.FatalAssert(len(rows) == 2, "Must be 2 rows in a table")
    
    _lib.FatalAssert(set(rows) == set(rows2), "COntents of tables on both nodes must be same")
    
    _lib.StartTestGroup("Check cancel of following transactions")
    
    # temporary stop second node
    startnode.StopNode(datadir2)
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row3'")
    
    startnode.StartNode(datadir2, address2,'30001', "Server 2")
    
    _sql.ExecuteSQLOnProxy(datadir2,"INSERT INTO test SET b='row3.2'")
    _sql.ExecuteSQLOnProxy(datadir,"UPDATE test SET b='row3_updated' WHERE a=3")
    _sql.ExecuteSQLOnProxy(datadir2,"UPDATE test SET b='row3_updated_2' WHERE a=3")
    
    _sql.ExecuteSQLOnProxy(datadir,"UPDATE test SET b='row3_updated_again' WHERE a=3")
    
    blocks = _blocks.WaitBlocks(datadir, 4)
    time.sleep(1)# give time to send transaction
    
    blocks = _blocks.WaitBlocks(datadir2, 4)
    time.sleep(1)# give time to send transaction
    
    rows = _lib.DBGetRows(datadir,"SELECT * FROM test ORDER BY a",True)
    
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table")
    
    rows2 = _lib.DBGetRows(datadir2,"SELECT * FROM test ORDER BY a",True)
    
    _lib.FatalAssert(len(rows2) == 3, "Must be 3 rows in a table")
    
    
    
    _lib.FatalAssert(set(rows) == set(rows2), "COntents of tables on both nodes must be same")
    
    startnode.StopNode(datadir)
    datadir = ""
    
    startnode.StopNode(datadir2)
    datadir2 = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    

    