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
    
    _lib.StartTestGroup("Init Blockchain on non empty DB")

    _lib.CleanTestFolders()
    datadir = _lib.CreateTestFolder()
    
    address2 = transactions.CreateWallet(datadir)
    address3 = transactions.CreateWallet(datadir)
    
    _lib.CopyTestConsensusConfig(datadir,"disabledcreate", address2)
    
    # add some data to the DB
    _lib.DBExecute(datadir,"create table test (a int unsigned auto_increment primary key, b varchar(10))")
    _lib.DBExecute(datadir,"insert into test SET b='row1'")
    _lib.DBExecute(datadir,"insert into test SET b='row2'")
    _lib.DBExecute(datadir,"create table members (a int unsigned auto_increment primary key, b varchar(10))")
    _lib.DBExecute(datadir,"insert into members SET b='row1'")
    _lib.DBExecute(datadir,"insert into members SET b='row2'")
    
    address = startnode.InitBockchain(datadir)
    _complex.AddProxyToConfig(datadir, "localhost:40041")
    _complex.AddInternalKeyToConfig(datadir, address3) # init internal signing

    startnode.StartNode(datadir, address, '30000')
    
    _lib.StartTestGroup("Do transactions")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blocks = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blocks) == 2,"Should be 2 blocks in blockchain")
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO members SET b='row3'")
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO members SET b='row4'")
    
    blocks = _blocks.WaitBlocks(datadir, 3)
    
    _lib.FatalAssert(len(blocks) == 3,"Should be 3 blocks in blockchain")
    
    time.sleep(1)
    
    
    _sql.ExecuteSQLOnProxyFail(datadir,"INSERT INTO test SET b='row3'")
    
    _transfers.Send(datadir,address, address3 ,1)
    
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row3'")
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO test SET b='row4'")
    _sql.ExecuteSQLOnProxy(datadir,"INSERT INTO members SET b='row5'")
    
    startnode.StopNode(datadir)
    datadir = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    

    