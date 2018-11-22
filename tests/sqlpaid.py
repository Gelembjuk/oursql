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

    startnode.StartNodeWithoutBlockchain(datadir)
    
    address2 = transactions.CreateWallet(datadir)
    address3 = transactions.CreateWallet(datadir)
    
    _lib.CopyTestConsensusConfig(datadir,"customnumbers", address2)
    
    address = startnode.InitBockchain(datadir)
    _complex.AddProxyToConfig(datadir, "localhost:40041")
    _complex.AddInternalKeyToConfig(datadir, address3) # init internal signing

    startnode.StartNode(datadir, address, '30000')
    
    _lib.StartTestGroup("Do transactions")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    # address3 signs proxy transactions. it doesn't have money yet
    _transfers.Send(datadir,address, address3 ,5) # needs this to create table
    
    blocks = _blocks.WaitBlocks(datadir, 2)
    time.sleep(1)
    
    _sql.ExecuteSQLOnProxy(datadir, "CREATE TABLE test (a INT auto_increment PRIMARY KEY, b VARCHAR(20))")
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row1'")
    
    blocks = _blocks.WaitBlocks(datadir, 3)
    time.sleep(1)
    
    # now try to create paid table. it should fail because no money
    _sql.ExecuteSQLOnProxyFail(datadir, "CREATE TABLE members (id INT auto_increment PRIMARY KEY, name VARCHAR(20))")
    
    # send money
    _transfers.Send(datadir,address, address3 ,10) # this table creation costs 10 
    
    _sql.ExecuteSQLOnProxy(datadir, "CREATE TABLE members (id INT auto_increment PRIMARY KEY, name VARCHAR(20))")
    
    tables = _lib.DBGetRows(datadir,"SHOW TABLES")
    found = False
    for table in tables:
        if table[0] == "members":
            found = True
            break
    
    _lib.FatalAssert(found, "Table not found in the DB")
    
    _sql.ExecuteSQLOnProxyFail(datadir, "INSERT INTO members SET name='user1'")
    # but should be abe to isert to free table
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row2'")
    
    blocks = _blocks.WaitBlocks(datadir, 4)
    time.sleep(1)
    
    _transfers.Send(datadir,address, address3 ,1) # this table creation costs 10 
    
    _sql.ExecuteSQLOnProxy(datadir, "INSERT INTO members SET name='user1'")
    _sql.ExecuteSQLOnProxy(datadir, "INSERT INTO members SET name='user2'")
    _sql.ExecuteSQLOnProxyFail(datadir, "INSERT INTO members SET name='user3'")
    
    bal1 = _transfers.GetBalance(datadir, address)
    bal2 = _transfers.GetBalance(datadir, address2)
    bal3 = _transfers.GetBalance(datadir, address3)
    
    _lib.FatalAssert(bal1[1] == 45.0, "Balance of a first addres is expected to be 45")
    _lib.FatalAssert(bal2[1] == 15.0, "Balance of a second addres is expected to be 15")
    _lib.FatalAssert(bal3[1] == 0.0, "Balance of a third addres is expected to be 0")
    
    startnode.StopNode(datadir)
    datadir = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    

    