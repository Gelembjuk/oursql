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
    
    _lib.StartTestGroup("SQL Consensus rules postponed")

    _lib.CleanTestFolders()
    datadir = _lib.CreateTestFolder()

    startnode.StartNodeWithoutBlockchain(datadir)
    
    address2 = transactions.CreateWallet(datadir)
    address3 = transactions.CreateWallet(datadir)
    
    _lib.CopyTestConsensusConfig(datadir,"postponedlimits", address2)
    
    address = startnode.InitBockchain(datadir)
    _complex.AddProxyToConfig(datadir, "localhost:40041")
    _complex.AddInternalKeyToConfig(datadir, address3) # init internal signing

    startnode.StartNode(datadir, address, '30000')
    
    _lib.StartTestGroup("Do transactions")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    _sql.ExecuteSQLOnProxy(datadir, "CREATE TABLE test (a INT auto_increment PRIMARY KEY, b VARCHAR(20))")
    
    blocks = _blocks.WaitBlocks(datadir, 2)
    time.sleep(1)
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row1'")
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row2'")
    
    blocks = _blocks.WaitBlocks(datadir, 3)
    time.sleep(1)
    
    _sql.ExecuteSQLOnProxy(datadir, "CREATE TABLE members (id INT auto_increment PRIMARY KEY, name VARCHAR(20))")
    
    tables = _lib.DBGetRows(datadir,"SHOW TABLES")
    found = False
    for table in tables:
        if table[0] == "members":
            found = True
            break
    
    _lib.FatalAssert(found, "Table not found in the DB")
    
    _sql.ExecuteSQLOnProxy(datadir, "INSERT INTO members SET name='user1'")
    
    _sql.ExecuteSQLOnProxy(datadir, "INSERT INTO members SET name='use2'")
    
    #_sql.ExecuteSQLOnProxy(datadir, "DROP TABLE members")
    
    blocks = _blocks.WaitBlocks(datadir, 4)
    time.sleep(1)
    
    _sql.ExecuteSQLOnProxyFail(datadir, "CREATE TABLE test2 (a INT auto_increment PRIMARY KEY, b VARCHAR(20))")
    
    _sql.ExecuteSQLOnProxy(datadir, "DROP TABLE members")
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row3'")
    _sql.ExecuteSQLOnProxy(datadir,"UPDATE test SET b='row3+upd1' WHERE a=3")
    _sql.ExecuteSQLOnProxy(datadir,"UPDATE test SET b='row3+upd2' WHERE a=3")
    
    blocks = _blocks.WaitBlocks(datadir, 5)
    time.sleep(1)
    # now this must be paid
    _sql.ExecuteSQLOnProxyFail(datadir,"UPDATE test SET b='row3+upd2' WHERE a=3")
    
    _sql.ExecuteSQLOnProxy(datadir, "CREATE TABLE members (id INT auto_increment PRIMARY KEY, name VARCHAR(20))")
    
    tables = _lib.DBGetRows(datadir,"SHOW TABLES")
    found = False
    for table in tables:
        if table[0] == "members":
            found = True
            break
        
    _sql.ExecuteSQLOnProxy(datadir, "INSERT INTO members SET name='user1'")
    _sql.ExecuteSQLOnProxy(datadir, "INSERT INTO members SET name='user2'")
    _sql.ExecuteSQLOnProxy(datadir, "INSERT INTO members SET name='user3'")
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row4'")
    
    blocks = _blocks.WaitBlocks(datadir, 6)
    time.sleep(1)
    
    _sql.ExecuteSQLOnProxyFail(datadir, "INSERT INTO members SET name='user4'")
    _sql.ExecuteSQLOnProxyFail(datadir,"INSERT INTO test SET b='row5'")
    
    # send money to be able to execute this
    _transfers.Send(datadir,address, address3 ,10) # needs this to create table
    _sql.ExecuteSQLOnProxy(datadir, "INSERT INTO members SET name='user4'")
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row5'")
    _sql.ExecuteSQLOnProxy(datadir, "CREATE TABLE test2 (a INT auto_increment PRIMARY KEY, b VARCHAR(20))")
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test2 SET b='row1'")
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test2 SET b='row2'")
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test2 SET b='row3'")
    
    blocks = _blocks.WaitBlocks(datadir, 7)
    time.sleep(1)
    
    startnode.StopNode(datadir)
    datadir = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    

    