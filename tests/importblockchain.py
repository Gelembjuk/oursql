import _lib
import _transfers
import _blocks
import _node
import re
import os
import time
import startnode
import initblockchain
import managenodes
import blocksbasic
import transactions

datadir1 = ""
datadir2 = ""

def allowgrouprun():
    return False

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
    
    _lib.StartTestGroup("Import long blockchain")

    _lib.CleanTestFolders()
    
    _lib.StartTestGroup("Copy blockchain from dataset")
    
    datadir1 = _lib.CreateTestFolder()
    datadir2 = _lib.CreateTestFolder()
    
    _lib.CopyTestData(datadir1,"bigchain")
    
    balances = _transfers.GetGroupBalance(datadir1)
    address = balances.keys()[0]
    
    startnode.StartNode(datadir1, address,'30000', "Server 1")

    address2 = initblockchain.ImportBockchain(datadir2,"localhost",'30000')
    
    #managenodes.RemoveAllNodes(datadir1)
    
    startnode.StartNode(datadir2, address2,'30001', "Server 2")
    
    #managenodes.RemoveAllNodes(datadir2)
    
    #managenodes.AddNode(datadir1, "localhost",'30001')
    
    blocks1 = _blocks.GetBlocks(datadir1)
    
    state = _node.NodeState(datadir1);
    
    _node.WaitBlocksInState(datadir2,len(blocks1), 120)
    
    blocks2 = _blocks.WaitBlocks(datadir2,len(blocks1), 120)
    
    #print len(blocks1), len(blocks2)
    
    _lib.FatalAssert(len(blocks1) == len(blocks2),"Number of bocks must be same on both servers")
    
    startnode.StopNode(datadir1,"Server 1")
    startnode.StopNode(datadir2,"Server 2")
    
    
    datadir1 = ""
    datadir2 = ""
    
    _lib.EndTestGroupSuccess()
        