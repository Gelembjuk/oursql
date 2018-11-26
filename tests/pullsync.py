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

datadir1 = ""
datadir2 = ""
datadir3 = ""

def aftertest(testfilter):
    global datadir1
    global datadir2
    global datadir3
    
    if datadir1 != "" or datadir2 != "" or datadir3 != "":
        _lib.StartTestGroup("Ending After failure of the test")
    
    if datadir1 != "":
        startnode.StopNode(datadir1,"Server 1")
    if datadir2 != "":    
        startnode.StopNode(datadir2,"Server 2")
    if datadir3 != "":    
        startnode.StopNode(datadir3,"Server 3")
        
def test(testfilter):
    global datadir1
    global datadir2
    global datadir3
    
    _lib.StartTestGroup("Test data exahcnge with non-public address node")
    
    _lib.CleanTestFolders()
    
    inf = blocksnodes.MakeBlockchainWithBlocks('30000')
    datadir = inf[0]
    address1 = inf[1]
    address1_2 = inf[2]
    address1_3 = inf[3]
    
    #_node.StartNodeInteractive(datadir, address1,'30000', "Server 1")
    startnode.StartNode(datadir, address1,'30000', "Server 1")
    datadir1 = datadir
    managenodes.RemoveAllNodes(datadir1)
    
    d = StartNodeAndImport('30001', '30000', "Server 2", 0, "_2_", "xxx.com" )
    datadir2 = d[0]
    address2 = d[1]
    
    d = StartNodeAndImport('30002', '30000', "Server 3", 0, "_3_")
    datadir3 = d[0]
    address3 = d[1]
    
    time.sleep(2)
    nodes = managenodes.GetNodes(datadir1)
    _lib.FatalAssert(len(nodes) == 2,"Should be 2 nodes on server 1")
    nodes = managenodes.GetNodes(datadir2)
    _lib.FatalAssert(len(nodes) == 2,"Should be 2 nodes on server 2")
    nodes = managenodes.GetNodes(datadir3)
    _lib.FatalAssert(len(nodes) == 2,"Should be 2 nodes on server 3")
    
    # make transaction on first node 
    tx=_transfers.Send(datadir,address1,address2,0.1)
    
    time.sleep(5)
    
    # this TX should appear on second node too
    txlist = transactions.GetUnapprovedTransactions(datadir2)
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
    
    tx=_transfers.Send(datadir2,address2,address1,0.01)
    
    txlist = transactions.GetUnapprovedTransactions(datadir2)
    _lib.FatalAssert(len(txlist) == 2,"Should be 2 unapproved transactions on second node")
    
    time.sleep(1)
    txlist = transactions.GetUnapprovedTransactions(datadir)
    _lib.FatalAssert(len(txlist) == 2,"Should be 2 unapproved transactions on first node")
    
    # make block on a secod node
    tx=_transfers.Send(datadir2,address2,address1,0.01)
    tx=_transfers.Send(datadir2,address2,address1,0.01)
    
    blocks = _blocks.WaitBlocks(datadir2, 5)
    _lib.FatalAssert(len(blocks) == 5,"Should be 5 blocks on server 2")
    
    time.sleep(1)
    
    blocks = _blocks.WaitBlocks(datadir1, 5)
    _lib.FatalAssert(len(blocks) == 5,"Should be 5 blocks on server 1")
    blocks = _blocks.WaitBlocks(datadir3, 5)
    _lib.FatalAssert(len(blocks) == 5,"Should be 5 blocks on server 3")
    
    time.sleep(1)
    # create block on the first node and check it appears on first
    for x in range(5):
        tx=_transfers.Send(datadir,address1,address2,0.1)
        
    blocks = _blocks.WaitBlocks(datadir1, 6)
    _lib.FatalAssert(len(blocks) == 6,"Should be 5 blocks on server 1")
    # extra transaction to pull
    tx=_transfers.Send(datadir,address1,address2,0.1)
    
    time.sleep(7)
    
    # new block should be pulled
    blocks = _blocks.WaitBlocks(datadir2, 6)
    _lib.FatalAssert(len(blocks) == 6,"Should be 6 blocks on server 2")
    
    txlist = transactions.GetUnapprovedTransactions(datadir1)
    
    time.sleep(2)
    
    txlist = transactions.GetUnapprovedTransactions(datadir2)

    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transactions on second node")
    
    startnode.StopNode(datadir1,"Server 1")
    datadir1 = ""
    
    startnode.StopNode(datadir2,"Server 2")
    datadir2 = ""
    
    startnode.StopNode(datadir3,"Server 3")
    datadir3 = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()

def StartNodeAndImport(port, importport, title, dbproxyport, suffix = "", host = "localhost"):
    
    datadir = _lib.CreateTestFolder(suffix)
    
    # this will create config file to remember other node address
    configfile = "{\"Port\": "+str(port)+",\"Nodes\":[{\"Host\": \"localhost\",\"Port\":"+str(importport)+"}]}"
    _lib.SaveConfigFile(datadir, configfile)
    
    address = initblockchain.ImportBockchain(datadir,"localhost",importport)
    
    _complex.AddMinterToConfig(datadir, address)
    
    if dbproxyport > 0:
        _complex.AddProxyToConfig(datadir, "localhost:"+str(dbproxyport))
        _complex.AddInternalKeyToConfig(datadir, address) # init internal signing
    
    startnode.StartNode(datadir, address, port, title, host)
    
    #check nodes. must be minimum 1 and import port must be present 
    nodes = managenodes.GetNodes(datadir)
    _lib.FatalAssert(len(nodes) > 0,"Should be minimum 1 nodes in output")
    
    return [datadir, address]


