import _lib
import _transfers
import _blocks
import re
import time
import startnode
import transactions
import random

#def beforetest(testfilter):
#    print "before test"
#def aftertest(testfilter):
    #print "after test"
def test(testfilter):
    _lib.StartTestGroup("Blocks making")
    
    _lib.CleanTestFolders()
    datadir = _lib.CreateTestFolder()
    
    r = PrepareBlockchain(datadir,'30000')
    address = r[0]

    # create another 3 addresses
    address2 = transactions.CreateWallet(datadir)
    address3 = transactions.CreateWallet(datadir)

    _lib.StartTestGroup("Do transactions")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    amount1 = '1'
    amount2 = '2'
    amount3 = '3'
    
    txid1 = _transfers.Send(datadir,address,address2,amount1)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
    
    time.sleep(1)
    txid2 = _transfers.Send(datadir,address,address3,amount2)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 2,"Should be 2 unapproved transaction")
    
    time.sleep(1)
    txid3 = _transfers.Send(datadir,address,address3,amount3)
    
    # node needs some time to make a block, so transaction still will be in list of unapproved
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 3,"Should be 3 unapproved transaction")
    
    time.sleep(1)
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
    
    _lib.FatalAssertFloat(amount1, txlist[txid1][2], "Amount of transaction 1 is wrong")
    
    _lib.FatalAssertFloat(amount2, txlist[txid2][2], "Amount of transaction 2 is wrong")
    
    _lib.FatalAssertFloat(amount3, txlist[txid3][2], "Amount of transaction 3 is wrong")
    
    _lib.FatalAssertFloat(amount1, txlist[txid4][2], "Amount of transaction 4 is wrong")
    
    blockchash = MintBlock(datadir,address)
    
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 2,"Should be 2 blocks in blockchain")
    
    _lib.StartTestGroup("Send 30 transactions")
    
    microamount = 0.01
    # send many transactions 
    for i in range(1,10):
        _lib.StartTest("Iteration "+str(i))
        txid1 = _transfers.Send(datadir,address,address2,microamount)
        txid2 = _transfers.Send(datadir,address2,address3,microamount)
        txid3 = _transfers.Send(datadir,address3,address,microamount)
        
        txlist = transactions.GetUnapprovedTransactions(datadir)
        
        _lib.FatalAssert(len(txlist) == i * 3,"Should be "+str(i*3)+" unapproved transaction")
        
        if txid1 not in txlist.keys():
            _lib.Fatal("Transaction 1 is not in the list of transactions after iteration "+str(i))
    
        if txid2 not in txlist.keys():
            _lib.Fatal("Transaction 2 is not in the list of transactions after iteration "+str(i))
    
        if txid3 not in txlist.keys():
            _lib.Fatal("Transaction 3 is not in the list of transactions after iteration "+str(i))
            
        time.sleep(1)
    
    blockchash = MintBlock(datadir,address)
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 3,"Should be 3 blocks in blockchain")
    
    _lib.StartTestGroup("Send 30 transactions. Random value")
    
    microamountmax = 0.01
    microamountmin = 0.0095
    # send many transactions 
    for i in range(1,11):
        _lib.StartTest("Iteration "+str(i))
        a1 = random.uniform(microamountmin, microamountmax)
        a2 = random.uniform(microamountmin, microamountmax)
        a3 = random.uniform(microamountmin, microamountmax)
        txid1 = _transfers.Send(datadir,address,address2,a1)
        txid2 = _transfers.Send(datadir,address2,address3,a2)
        txid3 = _transfers.Send(datadir,address3,address,a3)
        
        txlist = transactions.GetUnapprovedTransactions(datadir)
        
        _lib.FatalAssert(len(txlist) == i * 3,"Should be "+str(i*3)+" unapproved transaction")
        
        if txid1 not in txlist.keys():
            _lib.Fatal("Transaction 1 is not in the list of transactions after iteration "+str(i))
    
        if txid2 not in txlist.keys():
            _lib.Fatal("Transaction 2 is not in the list of transactions after iteration "+str(i))
    
        if txid3 not in txlist.keys():
            _lib.Fatal("Transaction 3 is not in the list of transactions after iteration "+str(i))
            
        time.sleep(1)
    
    blockchash = MintBlock(datadir,address)
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 4,"Should be 4 blocks in blockchain")
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    
def MintBlock(datadir,minter):
    _lib.StartTest("Force to Mint a block")
    res = _lib.ExecuteNode(['makeblock','-configdir',datadir,'-minter',minter,'-logs','trace'])
    _lib.FatalAssertSubstr(res,"New block mined with the hash","Block making failed")
    
    match = re.search( r'New block mined with the hash ([0-9a-zA-Z]+).', res)

    if not match:
        _lib.Fatal("New block hash can not be found in response "+res)
        
    blockhash = match.group(1)
    
    return blockhash

def PrepareBlockchain(datadir,port):
    _lib.StartTestGroup("Create blockchain. Try to start a server")
    
    _lib.StartTest("Try to start without blockchain")
    res = _lib.ExecuteNode(['startnode','-configdir',datadir])
    _lib.FatalAssertSubstr(res,"No database config","Blockchain is not yet inited. Should fail")
    
    _lib.StartTest("Create first address")
    res = _lib.ExecuteNode(['createwallet','-configdir',datadir])
    _lib.FatalAssertSubstr(res,"Your new address","Address creation returned wrong result")

    # get address from this response 
    match = re.search( r'.+: (.+)', res)

    if not match:
        _lib.Fatal("Address can not be found in "+res)
        
    address = match.group(1)
    
    dbconfig = _lib.GetDBCredentials(datadir)
    
    _lib.StartTest("Create blockchain")
    res = _lib.ExecuteNode(['initblockchain','-configdir',datadir, 
                            '-minter', address, 
                            '-mysqlhost', dbconfig['host'], 
                            '-mysqlport', dbconfig['port'],
                            '-mysqluser', dbconfig['user'],
                            '-mysqlpass', dbconfig['password'],
                            '-mysqldb', dbconfig['database'],
                            '-logs','trace'])
    _lib.FatalAssertSubstr(res,"Done!","Blockchain init failed")
    
    _lib.StartTest("Start normal")
    res = _lib.ExecuteNode(['startnode','-configdir',datadir,'-port',port,'-minter',address])
    _lib.FatalAssertStr(res,"","Should not be any output on succes start")

    # get process of the node. find this process exists
    _lib.StartTest("Check node state")
    res = _lib.ExecuteNode(['nodestate','-configdir',datadir])
    _lib.FatalAssertSubstr(res,"Server is running","Server should be runnning")

    # get address from this response 
    match = re.search( r'Process: (\d+),', res, re.M)

    if not match:
        _lib.Fatal("Can not get process ID from the response "+res)

    PID = int(match.group(1))

    _lib.FatalAssertPIDRunning(PID, "Can not find process with ID "+str(PID))

    _lib.StartTest("Start node again. should not be allowed")
    res = _lib.ExecuteNode(['startnode','-configdir',datadir])
    _lib.FatalAssertSubstr(res,"Already running or PID file exists","Second attempt to run should fail")
    
    _lib.StartTest("Check node state")
    res = _lib.ExecuteNode(['nodestate','-configdir',datadir])
    _lib.FatalAssertSubstr(res,"Server is running","Server should be runnning")

    # get address from this response 
    match = re.search( r'Process: (\d+),', res, re.M)

    if not match:
        _lib.Fatal("Can not get process ID from the response "+res)

    PID = int(match.group(1))
    
    _lib.FatalAssertPIDRunning(PID, "Can not find process with ID "+str(PID))
        
    _lib.StartTest("Stop node")
    res = _lib.ExecuteNode(['stopnode','-configdir',datadir])
    _lib.FatalAssert(res=="","Should not be any output on succes stop")

    time.sleep(1)
    _lib.FatalAssertPIDNotRunning(PID, "Process with ID "+str(PID)+" should not exist")
        
    _lib.StartTest("Stop node again")
    res = _lib.ExecuteNode(['stopnode','-configdir',datadir])
    _lib.FatalAssert(res=="","Should not be any output on succes stop")
    
    return [address,PID]

