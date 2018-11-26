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
    
    _lib.StartTestGroup("Run node with 30 blocks and test import from second node")
    
    _lib.CleanTestFolders()
    
    inf = blocksnodes.MakeBlockchainWithBlocks('30000')
    datadir = inf[0]
    address1 = inf[1]
    address1_2 = inf[2]
    address1_3 = inf[3]
    
    for x in range(5, 31):
        _lib.StartTestGroup("Create block #"+str(x))
        for y in range(0, x-1):
            tx=_transfers.Send(datadir,address1,address1_2,0.001)
        
        txlist = transactions.GetUnapprovedTransactions(datadir)
        _lib.FatalAssert(len(txlist) == x-1,"Should be "+str(x-1)+" unapproved transaction")
        
        blockchash = _blocks.MintBlock(datadir,address1)
        blocks = _blocks.WaitBlocks(datadir, x)
        _lib.FatalAssert(len(blocks) == x,"Should be "+str(x)+" blocks on server 1")
    
    
    #_node.StartNodeInteractive(datadir, address1,'30000', "Server 1")
    startnode.StartNode(datadir, address1,'30000', "Server 1")
    datadir1 = datadir
    managenodes.RemoveAllNodes(datadir1)
    
    d = pullsync.StartNodeAndImport('30001', '30000', "Server 2", 0, "_2_", "xxx.com" )
    datadir2 = d[0]
    address2 = d[1]
    
    blocks = _blocks.GetBlocks(datadir2)
    
    # wait when all 30 bocks are on node 2
    blocks = _blocks.WaitBlocks(datadir2, 30, 100)# 100 sec
    
    _lib.FatalAssert(len(blocks) == 30,"Should be 17 blocks on server 2")
    
    startnode.StopNode(datadir1,"Server 1")
    datadir1 = ""
    
    time.sleep(4)# allow to complete blocks adding
    startnode.StopNode(datadir2,"Server 2")
    datadir2 = ""
    
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


