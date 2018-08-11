import _lib
import _transfers
import re
import time
import startnode

#def beforetest(testfilter):
#    print "before test"
#def aftertest(testfilter):
#    print "after test"
def test(testfilter):
    _lib.StartTestGroup("Start/Stop node")

    _lib.CleanTestFolders()
    datadir = _lib.CreateTestFolder()

    startnode.StartNodeWithoutBlockchain(datadir)
    address = startnode.InitBockchain(datadir)
    startnode.StartNode(datadir, address, '30000')
    startnode.StopNode(datadir)
    
    # create another 3 addresses
    address2 = CreateWallet(datadir)
    address3 = CreateWallet(datadir)

    _lib.StartTestGroup("Do transactions")

    GetUnapprovedTransactionsEmpty(datadir)
    
    amount1 = '1'
    amount2 = '2'
    
    txid1 = _transfers.Send(datadir,address,address2,amount1)
    
    txlist = GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
        
    txid2 = _transfers.Send(datadir,address,address3,amount2)
    
    txlist = GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 2,"Should be 2 unapproved transaction")
    
    if txid1 not in txlist.keys():
        _lib.Fatal("Transaction 1 is not in the list of transactions")
    
    if txid2 not in txlist.keys():
        _lib.Fatal("Transaction 2 is not in the list of transactions")
    
    _lib.FatalAssertFloat(amount1, txlist[txid1][2], "Amount of transaction 1 is wrong")
    
    _lib.FatalAssertFloat(amount2, txlist[txid2][2], "Amount of transaction 2 is wrong")
    
    _lib.StartTestGroup("Cancel transaction")
    
    txid3 = _transfers.Send(datadir,address,address2,float(amount1)+float(amount2))
    
    txlist = GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 3,"Should be 3 unapproved transaction")
    
    CancelTransaction(datadir,txid3)
    
    txlist = GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 2,"Should be 2 unapproved transaction")
    
    #previous 2 transactions still must be in the list
    if txid1 not in txlist.keys():
        _lib.Fatal("Transaction 1 is not in the list of transactions")
    
    if txid2 not in txlist.keys():
        _lib.Fatal("Transaction 2 is not in the list of transactions")
    
    _transfers.SendTooMuch(datadir,address,address2,15)# the account should have only 10
    
    # cancel all transactions
    CancelTransaction(datadir,txid1)
    CancelTransaction(datadir,txid2)
    
    GetUnapprovedTransactionsEmpty(datadir)
    
    #==========================================================================
    # send when node server is running
    startnode.StartNode(datadir, address, '30000')
    
    txid1 = _transfers.Send(datadir,address,address2,amount1)
    
    txlist = GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
        
    txid2 = _transfers.Send(datadir,address,address3,amount2)
    
    txlist = GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 2,"Should be 2 unapproved transaction")
    
    if txid1 not in txlist.keys():
        _lib.Fatal("Transaction 1 is not in the list of transactions")
    
    if txid2 not in txlist.keys():
        _lib.Fatal("Transaction 2 is not in the list of transactions")
    
    startnode.StopNode(datadir)
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    
def CreateWallet(datadir):
    _lib.StartTestGroup("Create Wallet")
    
    _lib.StartTest("Create one more address")
    res = _lib.ExecuteNode(['createwallet','-configdir',datadir,'-logs','trace'])
    _lib.FatalAssertSubstr(res,"Your new address","Address creation returned wrong result")

    _lib.FatalRegex(r'.+: (.+)', res, "Address can not be found in "+res);
    
    # get address from this response 
    match = re.search( r'.+: (.+)', res)

    if not match:
        _lib.Fatal("Address can not be found in "+res)
        
    address = match.group(1)
    
    return address
    

def GetUnapprovedTransactionsEmpty(datadir):
    
    _lib.StartTest("Get unapproved transactions")
    res = _lib.ExecuteNode(['unapprovedtransactions','-configdir',datadir])
    _lib.FatalAssertSubstr(res,"Total transactions: 0","Output should not contains list of transactions")

def GetUnapprovedTransactions(datadir):
    
    _lib.StartTest("Get unapproved transactions")
    res = _lib.ExecuteNode(['unapprovedtransactions','-configdir',datadir])
    
    _lib.FatalAssertSubstr(res,"--- Transaction","Output should contains list of transactions")

    regex = ur"--- Transaction ([^:]+):"

    transactions = re.findall(regex, res)

    regex = ur"FROM ([A-Za-z0-9]+) TO ([A-Za-z0-9]+) VALUE ([0-9.]+)"
    
    txinfo = re.findall(regex, res)
    
    if len(txinfo) < len(transactions):
        regex = ur"SQL: ([^\n]+)\n"
        txinfo = re.findall(regex, res)
    
    txlist={}
    
    for i in range(len(transactions)):
        txlist[transactions[i]] = txinfo[i]
    
    return txlist

def CancelTransaction(datadir,txid):
    _lib.StartTest("Cancel transaction")
    
    res = _lib.ExecuteNode(['canceltransaction','-configdir',datadir,'-transaction',txid])
    
    _lib.FatalAssertSubstr(res,"Done!","Cancel of transaction failed")
    