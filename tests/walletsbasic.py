# tests all wallet operations with single node

import _lib
import _transfers
import _wallet
import _blocks
import re
import time
import math
import blocksnodes
import startnode
import transactions

datadir = ""

def aftertest(testfilter):
    global datadir
    
    if datadir != "":
        startnode.StopNode(datadir)
        
def test(testfilter):
    global datadir
    
    nodeport = '30000'
    
    _lib.StartTestGroup("Wallet Balance")
    
    _lib.CleanTestFolders()
    
    datadir_tmp = CopyBlockchainWithBlocks()
    
    balances = _transfers.GetGroupBalance(datadir_tmp)
    
    address1 = balances.keys()[0]
    address1_2 = balances.keys()[1]
    address1_3 = balances.keys()[2]
    
    balance = _transfers.GetBalance(datadir_tmp, address1)
    
    _lib.FatalAssert(balance[0] == balances[address1][0], "Balance is different from group result")
    # address1_3 becomes a minter. we will send money from other 2 and this will receive rewards
    startnode.StartNode(datadir_tmp, address1_3,nodeport)
    datadir = datadir_tmp
    
    blocks = _blocks.GetBlocks(datadir)
    blockslen = len(blocks)
    
    #create 2 wallet locations and 2 wallets in each of them
    walletdatadir1 = _lib.CreateTestFolder("wallet")
    walletdatadir2 = _lib.CreateTestFolder("wallet")
    
    addresses = _wallet.GetWallets(walletdatadir2)
    
    _lib.FatalAssert(len(addresses) == 0, "Expected 0 wallets")
    
    waddress1_1 = _wallet.CreateWallet(walletdatadir1);
    waddress1_2 = _wallet.CreateWallet(walletdatadir1);
    
    waddress2_1 = _wallet.CreateWallet(walletdatadir2);
    waddress2_2 = _wallet.CreateWallet(walletdatadir2);
    
    addresses = _wallet.GetWallets(walletdatadir2)
    
    _lib.FatalAssert(len(addresses) == 2, "Expected 2 wallets")
    
    addresses = _wallet.GetWallets(walletdatadir1)
    
    _lib.FatalAssert(len(addresses) == 2, "Expected 2 wallets for second folder")
    
    #send some funds to all that wallets
    amounttosend = "%.8f" % round(balances[address1][0]/8,8)
    
    # for next block minimum 6 TXt are required
    _transfers.Send(datadir,address1, waddress1_1 ,amounttosend)
    tx2_1 = _transfers.Send(datadir,address1, waddress1_2 ,amounttosend)
    _transfers.Send(datadir,address1, waddress2_1 ,amounttosend)
    _transfers.Send(datadir,address1, waddress1_1 ,amounttosend)
    _transfers.Send(datadir,address1, waddress1_1 ,amounttosend)
    _transfers.Send(datadir,address1, waddress1_1 ,amounttosend)
    
    # we control how blocks are created. here we wait on a block started and then send another 3 TX
    # we will get 2 more blocks here
    #blocks = _blocks.WaitBlocks(datadir, blockslen + 1)
    time.sleep(3)
    #_lib.FatalAssert(len(blocks) == blockslen + 1, "Expected "+str(blockslen + 1)+" blocks")
    
    # 7 TX are required  for next block
    _transfers.Send(datadir,address1, waddress2_2 ,amounttosend)
    
    amounttosend2 = "%.8f" % round(balances[address1_2][0]/8,8)
    
    _transfers.Send(datadir,address1_2, waddress1_1 ,amounttosend2)
    tx2_2 = _transfers.Send(datadir,address1_2, waddress1_2 ,amounttosend2)
    
    _transfers.Send(datadir,address1_2, waddress1_2 ,amounttosend2)
    _transfers.Send(datadir,address1_2, waddress1_2 ,amounttosend2)
    _transfers.Send(datadir,address1_2, waddress1_2 ,amounttosend2)
    _transfers.Send(datadir,address1_2, waddress1_2 ,amounttosend2)
    
    # wait to complete blocks 
    blocks = _blocks.WaitBlocks(datadir, blockslen + 2)
    time.sleep(5)    
    _lib.FatalAssert(len(blocks) == blockslen + 2, "Expected "+str(blockslen + 2)+" blocks")
    
    #get balances on wallets
    am1 = _wallet.GetBalanceWallet(walletdatadir1, waddress1_1, "localhost", nodeport)
    am2 = _wallet.GetBalanceWallet(walletdatadir1, waddress1_2, "localhost", nodeport)
    am3 = _wallet.GetBalanceWallet(walletdatadir2, waddress2_1, "localhost", nodeport)
    am4 = _wallet.GetBalanceWallet(walletdatadir2, waddress2_2, "localhost", nodeport)
    
    _lib.FatalAssert(am1[1] == round(float(amounttosend) * 4 + float(amounttosend2),8), "Expected balance is different for wallet 1_1")
    _lib.FatalAssert(am2[1] == round(float(amounttosend) + float(amounttosend2) * 5,8), "Expected balance is different for wallet 1_2")
    _lib.FatalAssert(am3[1] == float(amounttosend), "Expected balance is different for wallet 2_1")
    _lib.FatalAssert(am4[1] == float(amounttosend), "Expected balance is different for wallet 2_2")
    
    #get group blances on a wallet loc
    balances_new = _transfers.GetGroupBalance(datadir)
    
    #get balances on node wallets
    
    balances1 = _wallet.GetGroupBalanceWallet(walletdatadir1,"localhost", nodeport)
    balances2 = _wallet.GetGroupBalanceWallet(walletdatadir2,"localhost", nodeport)
    
    _lib.FatalAssert(am1[1] == balances1[waddress1_1][1], "Expected balance is different from group listing for 1_1")
    _lib.FatalAssert(am2[1] == balances1[waddress1_2][1], "Expected balance is different from group listing for 1_2")
    _lib.FatalAssert(am3[1] == balances2[waddress2_1][1], "Expected balance is different from group listing for 2_1")
    _lib.FatalAssert(am4[1] == balances2[waddress2_2][1], "Expected balance is different from group listing for 2_2")
    
    newbalance1 = round(balances[address1][0]  - float(amounttosend) * 7,8) 
    
    _lib.FatalAssert(newbalance1 == balances_new[address1][1], "Expected balance is different after spending")
    
    #send from wallets . 8 TXs
    _wallet.Send(walletdatadir1,waddress1_1, address1 ,amounttosend,"localhost", nodeport)
    _wallet.Send(walletdatadir1,waddress1_1, address1_2 ,amounttosend2,"localhost", nodeport)
    _wallet.Send(walletdatadir1,waddress1_2 ,address1, amounttosend,"localhost", nodeport)
    
    amounttosend3 = "%.8f" % round(float(amounttosend2)/8, 8)
    
    _wallet.Send(walletdatadir1,waddress1_2, address1_2 ,amounttosend3,"localhost", nodeport)
    _wallet.Send(walletdatadir1,waddress1_2, address1_2 ,amounttosend3,"localhost", nodeport)
    _wallet.Send(walletdatadir1,waddress1_2, address1_2 ,amounttosend3,"localhost", nodeport)
    _wallet.Send(walletdatadir1,waddress1_2, address1_2 ,amounttosend3,"localhost", nodeport)
    tx2_3 = _wallet.Send(walletdatadir1,waddress1_2, address1_2 ,amounttosend3,"localhost", nodeport)
    
    blocks = _blocks.WaitBlocks(datadir, blockslen + 3)
    _lib.FatalAssert(len(blocks) == blockslen + 3, "Expected "+str(blockslen + 3)+" blocks")
    time.sleep(2)
    
    am1_back = _wallet.GetBalanceWallet(walletdatadir1, waddress1_1, "localhost", nodeport)
    am1_expected = round(am1[1] - float(amounttosend) - float(amounttosend2),8)
    
    _lib.FatalAssert(am1_back[1] == am1_expected, "Expected balance after sending from wallet 1_1 is wrong: "+str(am1_back)+", expected "+str(am1_expected))
    
    am2_back = _wallet.GetBalanceWallet(walletdatadir1, waddress1_2, "localhost", nodeport)
    
    _wallet.SendTooMuch(walletdatadir1,waddress1_2, address1 ,str(am2_back[1] + 0.00000001),"localhost", nodeport)
    
    _lib.StartTestGroup("Node in config")
    
    _wallet.SetNodeConfig(walletdatadir1,"localhost", nodeport)
    
    balances1 = _wallet.GetGroupBalanceWalletNoNode(walletdatadir1)
    
    am1 = _wallet.GetBalanceWalletNoNode(walletdatadir1, waddress1_1)
    am2 = _wallet.GetBalanceWalletNoNode(walletdatadir1, waddress1_2)
    
    _lib.FatalAssert(am1[1] == balances1[waddress1_1][1], "Expected balance is different from group listing for 1_1")
    _lib.FatalAssert(am2[1] == balances1[waddress1_2][1], "Expected balance is different from group listing for 1_2")
    
    tx4 = _wallet.SendNoNode(walletdatadir1,waddress1_2, address1_2 ,str(round(am2[1]/2,8)))
    
    _lib.StartTestGroup("Unspent transactions")
    
    unspent = _wallet.GetUnspentNoNode(walletdatadir1,waddress1_2)
    
    txunspent = []
    
    for i in unspent:
        txunspent.append(i[2])
        
    _lib.FatalAssert(tx2_3 in txunspent, "Unspent TX in not in array of expected")
    
    unspent2 = _wallet.GetUnspent(walletdatadir2,waddress1_2,"localhost", nodeport)

    _lib.FatalAssert(len(unspent) == len(unspent2), "NNNumber of unspent TXs should be same. No config")
    
    txunspent = []
    
    for i in unspent2:
        txunspent.append(i[2])
    
    _lib.FatalAssert(tx2_3 in txunspent, "Unspent TX in not in array of expected. No config")
    
    _lib.StartTestGroup("Get wallet history")
    
    history = _wallet.GetHistoryNoNode(walletdatadir1,waddress1_2)
    
    _lib.FatalAssert(len(history) == 12, "Expected 3 records in a history")
    
    history = _wallet.GetHistory(walletdatadir2,waddress2_2,"localhost", nodeport)
    
    _lib.FatalAssert(len(history) == 1, "Expected 1 record in a history")
    
    _lib.StartTestGroup("Pending balance")
    
    # we cancel. we don't need it anymore. for next tests we need 0 TXs
    #transactions.CancelTransaction(datadir,tx4)
    
    am3 = _wallet.GetBalanceWallet(walletdatadir2, waddress2_1,"localhost", nodeport)
   
    addb1_2 = _transfers.GetBalance(datadir, address1_2)
   
    amounttosend2 = "%.8f" % round(am3[0]/2 - 0.00000001,8)
    
    _wallet.Send(walletdatadir2,waddress2_1, address1_2 ,amounttosend2,"localhost", nodeport)
    
    am3_2 = _wallet.GetBalanceWallet(walletdatadir2, waddress2_1,"localhost", nodeport)
   
    addb1_2_2 = _transfers.GetBalance(datadir, address1_2)
   
    _lib.FatalAssert(am3[1] == am3_2[1], "Approved balance should be unchanged")
    _lib.FatalAssert(round(am3[0] - float(amounttosend2),8) == am3_2[0] , "Total balance should be changed")
    _lib.FatalAssert(round(am3[2] - float(amounttosend2),8) == am3_2[2], "Pending balance should be changed")

    _lib.FatalAssert(round(addb1_2_2[2] - addb1_2[2],8) == round(am3[2] - am3_2[2],8), "Pending difference should be same")
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
   
    amounttosend2 = "%.8f" % round(am3[1]/2,8)
    
    _wallet.Send(walletdatadir2,waddress2_1, address1_2 ,amounttosend2,"localhost", nodeport)
    
    am3_3 = _wallet.GetBalanceWallet(walletdatadir2, waddress2_1,"localhost", nodeport)
   
    addb1_2_3 = _transfers.GetBalance(datadir, address1_2)
    
    _lib.FatalAssert(am3[1] == am3_3[1], "Approved balance should be unchanged")

    _lib.FatalAssert(math.fabs(round(am3_2[0] - float(amounttosend2),8) - am3_3[0]) <= 0.00000001, "Total balance should be changed")
    _lib.FatalAssert(math.fabs(round(am3_2[2] - float(amounttosend2) - am3_3[2],8)) <= 0.00000001, "Pending balance should be changed")

    _lib.FatalAssert(math.fabs(round(addb1_2_3[2] - addb1_2[2],8) - round(am3[2] - am3_3[2],8))  <= 0.00000002, "Pending difference should be same after 2 sends")
    
    startnode.StopNode(datadir)
    datadir = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    
def CopyBlockchainWithBlocks():
    datadir = _lib.CreateTestFolder()
    _lib.CopyTestData(datadir,"bcwith4blocks")
    
    return datadir