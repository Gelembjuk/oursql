import _lib
import _transfers
import _blocks
import _sql
import re
import time
import startnode
import _node
import _complex
import blocksbasic
import blocksnodes
import managenodes
import initblockchain
import transactions
import random

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
    
    _lib.StartTestGroup("Blocks exhange between nodes")
    
    _lib.CleanTestFolders()
    
    inf = MakeBlockchainWithBlocks('30000')
    datadir = inf[0]
    address1 = inf[1]
    address1_2 = inf[2]
    address1_3 = inf[3]
    
    #_node.StartNodeInteractive(datadir, address1,'30000', "Server 1")
    _complex.AddProxyToConfig(datadir, "localhost:40001")
    _complex.AddInternalKeyToConfig(datadir, address1) # init internal signing
    
    startnode.StartNode(datadir, address1,'30000', "Server 1")
    datadir1 = datadir
    managenodes.RemoveAllNodes(datadir1)
    
    d = blocksnodes.StartNodeAndImport('30001', '30000', "Server 2", 40002 ,'_2_')
    datadir2 = d[0]
    address2 = d[1]
    
    d = blocksnodes.StartNodeAndImport('30002', '30000', "Server 3", 40003 ,'_3_')
    datadir3 = d[0]
    address3 = d[1]
    
    time.sleep(1)
    nodes = managenodes.GetNodes(datadir1)
    _lib.FatalAssert(len(nodes) == 2,"Should be 2 nodes on server 1")
    nodes = managenodes.GetNodes(datadir2)
    _lib.FatalAssert(len(nodes) == 2,"Should be 2 nodes on server 2")
    nodes = managenodes.GetNodes(datadir3)
    _lib.FatalAssert(len(nodes) == 2,"Should be 2 nodes on server 3")
    
    # get balance 
    _sql.ExecuteSQLOnProxy(datadir1,"CREATE TABLE test (a INT auto_increment PRIMARY KEY, b VARCHAR(20))")
    _sql.ExecuteSQLOnProxy(datadir1,"INSERT INTO test SET b='row1'")
    _sql.ExecuteSQLOnProxy(datadir1,"INSERT INTO test SET a=2,b='row2'")
    _sql.ExecuteSQLOnProxy(datadir1,"INSERT INTO test (b) VALUES ('row3')")
    
    blocks = _blocks.WaitBlocks(datadir1, 5)
    _lib.FatalAssert(len(blocks) == 5,"Should be 5 blocks on server 1")
    blocks = _blocks.WaitBlocks(datadir2, 5)
    _lib.FatalAssert(len(blocks) == 5,"Should be 5 blocks on server 2")
    blocks = _blocks.WaitBlocks(datadir3, 5)
    _lib.FatalAssert(len(blocks) == 5,"Should be 5 blocks on server 3")
    
    time.sleep(1)# while all caches are cleaned
    
    managenodes.RemoveAllNodes(datadir1)
    managenodes.RemoveAllNodes(datadir2)
    managenodes.RemoveAllNodes(datadir3)
    
    rows = _lib.DBGetRows(datadir1,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table on node 1")
    rows = _lib.DBGetRows(datadir2,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table on node 2")
    rows = _lib.DBGetRows(datadir3,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table on node 3")
    
    _sql.ExecuteSQLOnProxy(datadir1,"INSERT INTO test (b) VALUES ('row4')")
    
    time.sleep(1)# while all caches are cleaned
    
    rows = _lib.DBGetRows(datadir1,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 4, "Must be 4 rows in a table on node 1")
    rows = _lib.DBGetRows(datadir2,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table on node 2")
    rows = _lib.DBGetRows(datadir3,"SELECT * FROM test",True)
    _lib.FatalAssert(len(rows) == 3, "Must be 3 rows in a table on node 3")
    
    startnode.StopNode(datadir1,"Server 1")
    datadir1 = ""
    
    startnode.StopNode(datadir2,"Server 2")
    datadir2 = ""
    
    startnode.StopNode(datadir3,"Server 3")
    datadir3 = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()


def MakeBlockchainWithBlocks(port):
    
    datadir = _lib.CreateTestFolder()
    
    r = blocksbasic.PrepareBlockchain(datadir,port)
    address = r[0]

    # create another 3 addresses
    address2 = transactions.CreateWallet(datadir)
    address3 = transactions.CreateWallet(datadir)

    _lib.StartTestGroup("Do _transfers")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    amount1 = '1'
    amount2 = '2'
    amount3 = '3'
    
    txid1 = _transfers.Send(datadir,address,address2,amount1)
    txlist = transactions.GetUnapprovedTransactions(datadir)
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
    
    txid2 = _transfers.Send(datadir,address,address3,amount2)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    _lib.FatalAssert(len(txlist) == 2,"Should be 2 unapproved transaction")
    txid3 = _transfers.Send(datadir,address,address3,amount3)
    
    # node needs some time to make a block, so transaction still will be in list of unapproved
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 3,"Should be 3 unapproved transaction")
    
    txid4 = _transfers.Send(datadir,address3,address2,amount1)
    
    # node needs some time to make a block, so transaction still will be in list of unapproved
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 4,"Should be 4 unapproved transaction")
    
    if txid1 not in txlist.keys():
        _lib.Fatal("Transaction 1 is not in the list of transactions")
    
    if txid2 not in txlist.keys():
        _lib.Fatal("Transaction 2 is not in the list of transactions")
    
    if txid3 not in txlist.keys():
        _lib.Fatal("Transaction 3 is not in the list of transactions")
    
    if txid4 not in txlist.keys():
        _lib.Fatal("Transaction 4 is not in the list of transactions")
    
    _lib.FatalAssertFloat(amount1, txlist[txid1][3], "Amount of transaction 1 is wrong")
    
    _lib.FatalAssertFloat(amount2, txlist[txid2][3], "Amount of transaction 2 is wrong")
    
    _lib.FatalAssertFloat(amount3, txlist[txid3][3], "Amount of transaction 3 is wrong")
    
    _lib.FatalAssertFloat(amount1, txlist[txid4][3], "Amount of transaction 4 is wrong")
    
    blockchash = _blocks.MintBlock(datadir,address)
    
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 2,"Should be 2 blocks in blockchain")
    
    _lib.StartTestGroup("Send 3 transactions")
    
    microamount = 0.01
    
    txid1 = _transfers.Send(datadir,address,address2,microamount)
    txid2 = _transfers.Send(datadir,address2,address3,microamount)
    txid3 = _transfers.Send(datadir,address3,address,microamount)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 3,"Should be 3 unapproved transaction")
    
    if txid1 not in txlist.keys():
        _lib.Fatal("Transaction 1 is not in the list of transactions after iteration "+str(i))

    if txid2 not in txlist.keys():
        _lib.Fatal("Transaction 2 is not in the list of transactions after iteration "+str(i))

    if txid3 not in txlist.keys():
        _lib.Fatal("Transaction 3 is not in the list of transactions after iteration "+str(i))
           
    blockchash = _blocks.MintBlock(datadir,address)
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 3,"Should be 3 blocks in blockchain")
    
    _lib.StartTestGroup("Send 3 transactions. Random value")
    
    microamountmax = 0.01
    microamountmin = 0.0095
    
    a1 = round(random.uniform(microamountmin, microamountmax),8)
    a2 = round(random.uniform(microamountmin, microamountmax),8)
    a3 = round(random.uniform(microamountmin, microamountmax),8)
    txid1 = _transfers.Send(datadir,address,address2,a1)
    txid2 = _transfers.Send(datadir,address2,address3,a2)
    txid3 = _transfers.Send(datadir,address3,address,a3)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 3,"Should be 3 unapproved transaction")
    
    if txid1 not in txlist.keys():
        _lib.Fatal("Transaction 1 is not in the list of transactions after iteration "+str(i))

    if txid2 not in txlist.keys():
        _lib.Fatal("Transaction 2 is not in the list of transactions after iteration "+str(i))

    if txid3 not in txlist.keys():
        _lib.Fatal("Transaction 3 is not in the list of transactions after iteration "+str(i))
        
    blockchash = _blocks.MintBlock(datadir,address)
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 4,"Should be 4 blocks in blockchain")
    
    return [datadir, address, address2, address3]
