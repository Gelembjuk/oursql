# tests pulling of data in case if a node doesn't have public address available and other nodes can not correct to it
import _lib
import _transfers
import _blocks
import re
import time
import startnode
import _node
import _complex
import blocksbasic
import managenodes
import initblockchain
import transactions
import random
import blocksnodes
import pullsync

datadir1 = ""
datadir2 = ""

def aftertest(testfilter):
    global datadir1
    global datadir2
    
    if datadir1 != "" or datadir2 != "":
        _lib.StartTestGroup("Ending After failure of the test")
    
    if datadir1 != "":
        startnode.StopNode(datadir1,"Server 1")
    if datadir2 != "":    
        startnode.StopNode(datadir2,"Server 2")
        
def test(testfilter):
    global datadir1
    global datadir2
    
    _lib.StartTestGroup("Test data exahcnge with non-public address node and long minting")
    
    _lib.CleanTestFolders()
    
    datadir = _lib.CreateTestFolder()

    address1_2 = transactions.CreateWallet(datadir)
    address1_3 = transactions.CreateWallet(datadir)
    
    _lib.CopyTestConsensusConfig(datadir,"hardpow", address1_3)
    
    address1_1 = startnode.InitBockchain(datadir)
    _complex.AddProxyToConfig(datadir, "localhost:40041")
    _complex.AddInternalKeyToConfig(datadir, address1_2) # init internal signing

    startnode.StartNode(datadir, address1_1, '30000')
    datadir1 = datadir
    
    tx=_transfers.Send(datadir,address1_1,address1_2,2)
    
    # node 1 should start minting now
    
    d = pullsync.StartNodeAndImport('30001', '30000', "Server 2", 0, "_2_", "xxx.com" )
    datadir2 = d[0]
    address2 = d[1]
    
    # list of transactions must be emoty on 2-nd node
    transactions.GetUnapprovedTransactionsEmpty(datadir2)
    time.sleep(1)
    transactions.GetUnapprovedTransactionsEmpty(datadir2)
    time.sleep(1)
    transactions.GetUnapprovedTransactionsEmpty(datadir2)
    time.sleep(1)
    
    # wait 2-nd block 
    blocks = _blocks.WaitBlocks(datadir2, 2)
    _lib.FatalAssert(len(blocks) == 2,"Should be 2 blocks on server 2")
    
    # 2 new TX to make next block
    tx=_transfers.Send(datadir,address1_1,address1_2,1)
    tx=_transfers.Send(datadir,address1_1,address1_2,1)
    time.sleep(1)
    tx=_transfers.Send(datadir,address1_1,address1_2,1)
    
    # on second node only the last TX should appear soon
    txlist = _transfers.WaitUnapprovedTransactions(datadir2, 1, 6)
    
    # there can be 2 cases. this TX can be based on other TX which is currently under building of a block
    # in this case TX will fail on pull
    if (len(txlist) == 0):
        # wait while 3-rd block appears, after this TX should be added
        blocks = _blocks.WaitBlocks(datadir2, 3)
        _lib.FatalAssert(len(blocks) == 3,"Should be 3 blocks on server 2")
        
        txlist = _transfers.WaitUnapprovedTransactions(datadir2, 1, 6)
    
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 transaction on server 2")
    
    _lib.FatalAssert(tx in txlist.keys(),"3-rd TX shoul be in keys")
    
    startnode.StopNode(datadir1,"Server 1")
    datadir1 = ""
    
    startnode.StopNode(datadir2,"Server 2")
    datadir2 = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()


