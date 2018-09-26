import _lib
import _transfers
import _blocks
import _complex
import _node
import _sql
import re
import os
import time
import random
import startnode
import blocksnodes
import managenodes
import transactions

datadirs = []

def aftertest(testfilter):
    global datadirs
    
    for datadir in datadirs:
        if datadir != "":
            startnode.StopNode(datadir)
        
def test(testfilter):
    global datadirs
    _lib.CleanTestFolders()
    #return _complex.PrepareNodesWithSQL()

    _lib.StartTestGroup("Load ready BC into 6 nodes")
    
    dirs = _complex.Copy6NodesSQL()
    
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
    _lib.StartTestGroup("Check blockchain before updates")
    blocks1 = _blocks.GetBlocks(nodes[0]["datadir"])
    blocks2 = _blocks.GetBlocks(nodes[1]["datadir"])
    
    _lib.FatalAssert(len(blocks1) == 9,"First branch should have 9 blocks")
    _lib.FatalAssert(len(blocks2) == 8,"Second branch should have 8 blocks")
    
    _lib.FatalAssert(blocks1[2] == blocks2[1],"7 block must be same for both")
    _lib.FatalAssert(blocks1[1] != blocks2[0],"8 block must be different")
    
    _lib.StartTestGroup("Connect subnetworks")
    managenodes.AddNode(nodes[0]["datadir"],"localhost",'30001')
   
    # wait while blocks are exachanged
    _blocks.WaitBlocks(nodes[1]["datadir"],9)
    
    s = 1
    time.sleep(2)
    
    while s < 10:
        rows1 = _lib.DBGetRows(nodes[1]['datadir'],"SELECT * FROM test",True)
        rows2 = _lib.DBGetRows(nodes[2]['datadir'],"SELECT * FROM test",True)
        rows3 = _lib.DBGetRows(nodes[3]['datadir'],"SELECT * FROM test",True)
        
        if rows1[1][1] == "row2_updated" and rows2[1][1] == "row2_updated" and rows3[1][1] == "row2_updated":
            break
        
        time.sleep(1)
        s = s + 1
    
    # get unapproved transactions (after block cancel)
    txlist = _transfers.WaitUnapprovedTransactions(nodes[1]["datadir"], 3, 15)
    _lib.FatalAssert(len(txlist) == 3,"SHould be 3 unapproved TXs")
    
    #send another 2 TXs to have 9 required TXs
    balances = _transfers.GetGroupBalance(nodes[1]["datadir"])
    
    mainbalance = _transfers.GetGroupBalance(nodes[0]["datadir"])
    addr1 = balances.keys()[0]
    amount = "%.8f" % round(balances[addr1][0]/5,8)
    
    # add yet mode 6 TXs to complete a block
    _transfers.Send(nodes[1]["datadir"],addr1,mainbalance.keys()[0],amount)
    _transfers.Send(nodes[1]["datadir"],addr1,mainbalance.keys()[0],amount)
    
    # 4 SQL txs
    _sql.ExecuteSQLOnProxy(nodes[1]['datadir'],"INSERT INTO test SET b='row11', a=11")
    # on another node. should go to the second node too
    _sql.ExecuteSQLOnProxy(nodes[0]['datadir'],"INSERT INTO test SET b='row12', a=12")
    _sql.ExecuteSQLOnProxy(nodes[1]['datadir'],"UPDATE test set b='row5_update_other' WHERE a=5")
    
    time.sleep(2)
    
    txlist = _transfers.WaitUnapprovedTransactions(nodes[0]["datadir"], 2, 10)
    # there should be minimum 2 tx, maximum 4. Because currency TX's can be based on other 2 currency TX's that are not yet on other nodes (from canceled block)
    _lib.FatalAssert(len(txlist) >= 2 and len(txlist) <= 4,"SHould be from 2 to 4 unapproved TXs on node 1")
    
    txlist = _transfers.WaitUnapprovedTransactions(nodes[1]["datadir"], 8, 10)

    _lib.FatalAssert(len(txlist) == 8,"SHould be 8 unapproved TXs")
    
    # after this TX new block should be created
    _sql.ExecuteSQLOnProxy(nodes[1]['datadir'],"UPDATE test set b='row11_update_other' WHERE a=11")
    
    # wait while new block created and posted to all other
    b1 = _blocks.WaitBlocks(nodes[1]["datadir"],10)
    b2 = _blocks.WaitBlocks(nodes[0]["datadir"],10)
    
    _lib.StartTestGroup("Check blockchain after updates")
    blocks2_0 = _blocks.GetBlocks(nodes[0]["datadir"])

    _lib.FatalAssert(len(blocks2_0) == 10,"10 block must be on node 0")
    _lib.FatalAssert(blocks2_0[1] == blocks1[0],"9 block must be same for both")
    
    blocks2_1 = _blocks.GetBlocks(nodes[1]["datadir"])
    _lib.FatalAssert(len(blocks2_1) == 10,"10 block must be on node 1")
    _lib.FatalAssert(blocks2_1[1] == blocks1[0],"9 block must be same for both")
        
    _lib.StartTestGroup("Node 2 "+os.path.basename(nodes[2]["datadir"]))
    _blocks.WaitBlocks(nodes[2]["datadir"],10)
    
    blocks2_2 = _blocks.GetBlocks(nodes[2]["datadir"])
    
    _lib.FatalAssert(len(blocks2_2) == 10,"10 block must be on node 2")
    _lib.FatalAssert(blocks2_2[1] == blocks2_1[1],"2-nd from top blocks on 2 must be same as on 1")
    
    _lib.StartTestGroup("Node 3 "+os.path.basename(nodes[3]["datadir"]))
    
    _blocks.WaitBlocks(nodes[3]["datadir"],10)
    blocks2_3 = _blocks.GetBlocks(nodes[3]["datadir"])
    
    _lib.FatalAssert(len(blocks2_3) == 10,"10 block must be on node 3")
    _lib.FatalAssert(blocks2_3[1] == blocks2_1[1],"2-nd from top blocks on  3 must be same as on 1")
    
    _lib.StartTestGroup("Node 4 "+os.path.basename(nodes[4]["datadir"]))
    
    _blocks.WaitBlocks(nodes[4]["datadir"],10)
    blocks2_4 = _blocks.GetBlocks(nodes[4]["datadir"])
    
    _lib.FatalAssert(len(blocks2_4) == 10,"10 block must be on node 4")
    _lib.FatalAssert(blocks2_4[1] == blocks2_1[1],"2-nd from top blocks on 4 must be same as on 1")
    
    _lib.StartTestGroup("Node 5 "+os.path.basename(nodes[5]["datadir"]))
    
    _blocks.WaitBlocks(nodes[5]["datadir"],10)
    blocks2_5 = _blocks.GetBlocks(nodes[5]["datadir"])
    
    _lib.FatalAssert(len(blocks2_5) == 10,"10 block must be on node 5")
    _lib.FatalAssert(blocks2_5[1] == blocks2_1[1],"2-nd from top blocks on 5 must be same as on 1")
    
    rows1 = _lib.DBGetRows(nodes[0]['datadir'],"SELECT * FROM test",True)
    rows2 = _lib.DBGetRows(nodes[1]['datadir'],"SELECT * FROM test",True)
    
    _lib.FatalAssert(rows1 == rows2,"Table contents on node 1 and 2 must be same")
    
    rows3 = _lib.DBGetRows(nodes[2]['datadir'],"SELECT * FROM test",True)
    
    _lib.FatalAssert(rows1 == rows3,"Table contents on node 1 and 3 must be same")
    
    rows4 = _lib.DBGetRows(nodes[3]['datadir'],"SELECT * FROM test",True)
    
    _lib.FatalAssert(rows1 == rows4,"Table contents on node 1 and 4 must be same")
    
    rows5 = _lib.DBGetRows(nodes[4]['datadir'],"SELECT * FROM test",True)
    
    _lib.FatalAssert(rows1 == rows5,"Table contents on node 1 and 5 must be same")
    
    rows6 = _lib.DBGetRows(nodes[5]['datadir'],"SELECT * FROM test",True)
    
    _lib.FatalAssert(rows1 == rows6,"Table contents on node 1 and 6 must be same")
    
    _lib.StartTestGroup("Final checks")
    # should be empty list of transactions now
    transactions.GetUnapprovedTransactionsEmpty(nodes[1]["datadir"])
    
    for node in nodes:
        startnode.StopNode(node['datadir'])
        datadirs[node['index']] = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()


