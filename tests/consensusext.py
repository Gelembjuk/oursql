import _lib
import _transfers
import _blocks
import _complex
import _node
import re
import os
import time
import random
import startnode
import blocksnodes
import managenodes
import transactions

datadirs = []

def allowgrouprun():
    return False

def aftertest(testfilter):
    global datadirs
    
    for datadir in datadirs:
        if datadir != "":
            startnode.StopNode(datadir)
        
def test(testfilter):
    global datadirs
    _lib.CleanTestFolders()
    #return _complex.Make5BlocksBC()
    #return _complex.PrepareNodes()

    dirs = _complex.Copy6Nodes()
    
    nodes = []
    
    i = 1
    for d in dirs:
        #get address in wallets in this dir 
        balances = _transfers.GetGroupBalance(d)
        address = balances.keys()[0]
        
        port = str(30000 + i )
        
        nodes.append({'index':i - 1, 'port':port, 'datadir':d,'address':address,"title":"Server "+str(i)})
        
        #_transfers.ReindexUTXO(d)
        #txlist = transactions.GetUnapprovedTransactions(d)
        #print txlist
        #start this node 
        #print os.path.basename(d)
        startnode.StartNodeConfig(d)
        
        i = i + 1
        datadirs.append(d)
        
    #check nodes on each node is correct 
    for node in nodes:
        #print os.path.basename(node['datadir'])
        nodeslist = managenodes.GetNodes(node['datadir'])
        
        _lib.FatalAssert(len(nodeslist) == 2,"Should be 2 nodes on "+node["title"])
        
        if node['index'] == 0:
            _lib.FatalAssert("localhost:30005" in nodeslist,"Node 6 should be on the node 0")
            _lib.FatalAssert("localhost:30004" in nodeslist,"Node 5 should be on the node 0")
        if node['index'] == 1:
            _lib.FatalAssert("localhost:30002" in nodeslist,"Node 2 should be on the node 1")
            _lib.FatalAssert("localhost:30003" in nodeslist,"Node 3 should be on the node 1")
    #raise ValueError('Stop')
    # test blocks branches
    # ensure subnetworks are fine
    managenodes.AddNode(nodes[2]["datadir"],"localhost",'30003')
    managenodes.AddNode(nodes[4]["datadir"],"localhost",'30005')
    
    _lib.StartTestGroup("Check blockchain before updates")
    blocks1 = _blocks.GetBlocks(nodes[0]["datadir"])
    blocks2 = _blocks.GetBlocks(nodes[1]["datadir"])
    
    _lib.FatalAssert(len(blocks1) == 9,"First branch should have 9 blocks")
    _lib.FatalAssert(len(blocks2) == 8,"Second branch should have 8 blocks")
    
    _lib.FatalAssert(blocks1[2] == blocks2[1],"7 block must be same for both")
    _lib.FatalAssert(blocks1[1] != blocks2[0],"8 block must be different")
    
    #======================================================================================
    # remove node 6 from the first branch
    managenodes.RemoveAllNodes(nodes[5]["datadir"])
    nodeslist = managenodes.GetNodes(nodes[5]["datadir"])
    _lib.FatalAssert(len(nodeslist) == 0,"Should be 0 nodes on the node 6")
    
    # remove this node from 2 other nodes where it is known
    managenodes.RemoveNode(nodes[0]["datadir"], "localhost" ,"30005")
    managenodes.RemoveNode(nodes[4]["datadir"], "localhost" ,"30005")
    
    nodeslist = managenodes.GetNodes(nodes[0]["datadir"])
    _lib.FatalAssert(len(nodeslist) == 1,"Should be 1 nodes on the node 1")
    
    nodeslist = managenodes.GetNodes(nodes[4]["datadir"])
    _lib.FatalAssert(len(nodeslist) == 1,"Should be 1 nodes on the node 5")
    
    # add one more blockon the first node
    balances = _transfers.GetGroupBalance(nodes[0]["datadir"])
    
    addr1 = balances.keys()[0]
    addr2 = balances.keys()[1]
    amount = "%.8f" % round(balances[addr1][0]/12,8)
    
    for x in range(1, 11):
        _transfers.Send(nodes[0]["datadir"],addr1,addr2,amount)
        
    _blocks.WaitBlocks(nodes[0]["datadir"],10)
    
    time.sleep(4)
    # and again new block.
    balances = _transfers.GetGroupBalance(nodes[0]["datadir"])
    
    addr1 = balances.keys()[1]
    addr2 = balances.keys()[0]
    amount = "%.8f" % round(balances[addr1][0]/12,8)
    
    for x in range(1, 12):
        _transfers.Send(nodes[0]["datadir"],addr1,addr2,amount)
        
    _blocks.WaitBlocks(nodes[0]["datadir"],11)
    
    time.sleep(4)
    # create 2 more blocks on branch 2
    balances = _transfers.GetGroupBalance(nodes[1]["datadir"])
    
    addr3 = balances.keys()[0]
    amount = "%.8f" % round(balances[addr3][0]/30,8)
    
    for x in range(1, 10):
        # send to address on the firs node
        _transfers.Send(nodes[1]["datadir"],addr3,addr1,amount)
        
    _blocks.WaitBlocks(nodes[1]["datadir"],9)
    
    time.sleep(4)
    
    for x in range(1, 11):
        # send to address on the firs node
        _transfers.Send(nodes[1]["datadir"],addr3,addr1,amount)
        
    _blocks.WaitBlocks(nodes[1]["datadir"],10)
    
    # now branch 2 has more blocks than the node 6.
    # connect node 6 with the branch 2
    _lib.StartTestGroup("Connect network 2 with the node 6")
    managenodes.AddNode(nodes[1]["datadir"],"localhost",'30005')
    managenodes.AddNode(nodes[4]["datadir"],"localhost",'30002')
    managenodes.AddNode(nodes[4]["datadir"],"localhost",'30003')
    managenodes.AddNode(nodes[4]["datadir"],"localhost",'30005')
    
    managenodes.WaitNodes(nodes[1]["datadir"],3)
    managenodes.WaitNodes(nodes[2]["datadir"],3)
    managenodes.WaitNodes(nodes[3]["datadir"],3)
    
    # should be 10 blocks after sync
    _blocks.WaitBlocks(nodes[5]["datadir"],10)
    
    # must be unapproved transactions from previous block 8
    #txlist = transactions.GetUnapprovedTransactions(nodes[5]["datadir"])
    _lib.StartTestGroup("Wait 11 blocks on every node of net 2")
    # new block is created from free transaction after reset of some blocks
    _blocks.WaitBlocks(nodes[5]["datadir"],11)
    _blocks.WaitBlocks(nodes[1]["datadir"],11)
    _blocks.WaitBlocks(nodes[2]["datadir"],11)
    _blocks.WaitBlocks(nodes[3]["datadir"],11)
    
    transactions.GetUnapprovedTransactionsEmpty(nodes[5]["datadir"])
    
    
    #at this point branch 2 + node 6 have 11 blocks.
    #branch 1 has also 11
    _lib.StartTestGroup("Connect all nodes")
    managenodes.AddNode(nodes[1]["datadir"],"localhost",'30000')
    #wait while all nodes know about other nodes
    
    netnodes = managenodes.WaitNodes(nodes[0]["datadir"],5)
    
    netnodes = managenodes.WaitNodes(nodes[1]["datadir"],5)
    
    netnodes = managenodes.WaitNodes(nodes[2]["datadir"],5)
    
    netnodes = managenodes.WaitNodes(nodes[3]["datadir"],5)
    
    netnodes = managenodes.WaitNodes(nodes[4]["datadir"],5)
    
    netnodes = managenodes.WaitNodes(nodes[5]["datadir"],5)
    
    nodeslist = managenodes.GetNodes(nodes[1]["datadir"])
    
    _lib.FatalAssert(len(nodeslist) == 5,"Should be 5 nodes on the node 2")
    
    # add more transactions on the node 0 to get 12-th block
    balances = _transfers.GetGroupBalance(nodes[0]["datadir"])
    
    addr1 = balances.keys()[0]
    addr2 = balances.keys()[2]
    amount = "%.8f" % round(balances[addr1][0]/14,8)
    
    # here a node can have some already prepared transactions . we will add 12. but total can be more
    
    for x in range(1, 13):
        tx = _transfers.Send(nodes[0]["datadir"],addr1,addr2,amount)
        
    _blocks.WaitBlocks(nodes[0]["datadir"],12)
   
    # now sync should start. branch 2 will change branch
    _blocks.WaitBlocks(nodes[1]["datadir"],12)
    
    _lib.StartTestGroup("Wait final blocks")
    
    # after some time new block should be created from all unconfirmed transactions
    blocks2_0 = _blocks.WaitBlocks(nodes[1]["datadir"],13)
    
    if len(blocks2_0) < 13:
        # send some more transactions. it can be after all consensus some previous tranactions are not valid anymore
        balances = _transfers.GetGroupBalance(nodes[5]["datadir"])
        addr1 = balances.keys()[0]
        amount = "%.8f" % round(balances[addr1][0]/14,8)
        for x in range(1, 11):
            tx = _transfers.Send(nodes[5]["datadir"],addr1,addr2,amount)
            
            
        blocks2_0 = _blocks.WaitBlocks(nodes[1]["datadir"],13)
    
    _lib.FatalAssert(len(blocks2_0) == 13,"13 block must be on node 0")
    _blocks.WaitBlocks(nodes[0]["datadir"],13)
    
    _lib.StartTestGroup("Check blockchain after updates")
    blocks2_0 = _blocks.GetBlocks(nodes[0]["datadir"])

    _lib.FatalAssert(len(blocks2_0) == 13,"13 block must be on node 0")
    
    blocks2_1 = _blocks.GetBlocks(nodes[1]["datadir"])
    _lib.FatalAssert(len(blocks2_1) == 13,"13 block must be on node 1")
        
    _lib.StartTestGroup("Node 2 "+os.path.basename(nodes[2]["datadir"]))
    _blocks.WaitBlocks(nodes[2]["datadir"],13)
    
    blocks2_2 = _blocks.GetBlocks(nodes[2]["datadir"])
    
    _lib.FatalAssert(len(blocks2_2) == 13,"13 block must be on node 2")
    #_lib.FatalAssert(blocks2_2[1] == blocks2_1[1],"2-nd from top blocks on 2 must be same as on 1")
    
    _lib.StartTestGroup("Node 3 "+os.path.basename(nodes[3]["datadir"]))
    
    _blocks.WaitBlocks(nodes[3]["datadir"],13)
    blocks2_3 = _blocks.GetBlocks(nodes[3]["datadir"])
    
    _lib.FatalAssert(len(blocks2_3) == 13,"13 block must be on node 3")
    #_lib.FatalAssert(blocks2_3[1] == blocks2_1[1],"2-nd from top blocks on  3 must be same as on 1")
    
    _lib.StartTestGroup("Node 4 "+os.path.basename(nodes[4]["datadir"]))
    
    _blocks.WaitBlocks(nodes[4]["datadir"],13)
    blocks2_4 = _blocks.GetBlocks(nodes[4]["datadir"])
    
    _lib.FatalAssert(len(blocks2_4) == 13,"13 block must be on node 4")
    #_lib.FatalAssert(blocks2_4[1] == blocks2_1[1],"2-nd from top blocks on 4 must be same as on 1")
    
    _lib.StartTestGroup("Node 5 "+os.path.basename(nodes[5]["datadir"]))
    
    _blocks.WaitBlocks(nodes[5]["datadir"],13)
    blocks2_5 = _blocks.GetBlocks(nodes[5]["datadir"])
    
    _lib.FatalAssert(len(blocks2_5) == 13,"13 block must be on node 5")
    #_lib.FatalAssert(blocks2_5[1] == blocks2_1[1],"2-nd from top blocks on 5 must be same as on 1")
    
    _lib.StartTestGroup("Final checks")
    
    # should be empty list of transactions now
    # we commented because it is not always empty and it is not bad
    #transactions.GetUnapprovedTransactionsEmpty(nodes[1]["datadir"])
    
    for node in nodes:
        startnode.StopNode(node['datadir'])
        datadirs[node['index']] = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()


