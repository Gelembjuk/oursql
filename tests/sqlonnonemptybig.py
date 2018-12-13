import _lib
import _transfers
import _complex
import _sql
import _blocks
import sys
import re
import time
import startnode
import transactions

datadir = ""

def allowgrouprun():
    return False

def aftertest(testfilter):
    global datadir
    
    if datadir != "":
        startnode.StopNode(datadir)
        
def test(testfilter):
    global datadir
    
    _lib.StartTestGroup("Init Blockchain on non empty DB. Add 4000 records")

    _lib.CleanTestFolders()
    datadir = _lib.CreateTestFolder()
    
    address2 = transactions.CreateWallet(datadir)
    address3 = transactions.CreateWallet(datadir)
    
    _lib.CopyTestConsensusConfig(datadir,"disabledcreate", address2)
    
    # add some data to the DB
    _lib.DBExecute(datadir,"create table test (a int unsigned auto_increment primary key, b varchar(10))")
    # add 2k records
    for x in range(100):
        _lib.DBExecute(datadir,"insert into test SET b='row"+str(x)+"'")
        if x%50 == 0:
            sys.stdout.write('.')
        if x%200 == 0 and x > 0:
            sys.stdout.write("-"+str(x)+"-")
    print("")
    _lib.DBExecute(datadir,"create table members (a int unsigned auto_increment primary key, b varchar(10))")
    for x in range(100):
        _lib.DBExecute(datadir,"insert into members SET b='row"+str(x)+"'")
        if x%20 == 0:
            sys.stdout.write('.')
        if x%200 == 0 and x > 0:
            sys.stdout.write("-"+str(x)+"-")
    print("")
    
    _lib.StartTestGroup("Data added. Init BC")
    
    address = startnode.InitBockchain(datadir)
    _complex.AddProxyToConfig(datadir, "localhost:40041")
    _complex.AddInternalKeyToConfig(datadir, address3) # init internal signing

    startnode.StartNode(datadir, address, '30000')
    
    _lib.StartTestGroup("Do transactions")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blocks = _blocks.GetBlocks(datadir)
    #print(blocks)
    #_lib.FatalAssert(len(blocks) == 4,"Should be 4 blocks in blockchain")
    _lib.FatalAssert(len(blocks) == 2,"Should be 2 blocks in blockchain")
    
    startnode.StopNode(datadir)
    datadir = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    

    