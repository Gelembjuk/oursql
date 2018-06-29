import _lib
import _wallet
import _transfers
import _blocks
import re
import time
import blocksnodes
import startnode

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
    
    inf = blocksnodes.MakeBlockchainWithBlocks(nodeport)
    datadir_tmp = inf[0]
    address1 = inf[1]
    address1_2 = inf[2]
    address1_3 = inf[3]
    
    balances = _transfers.GetGroupBalance(datadir_tmp)
    
    #get balance when a node is not run
    bal1 = _transfers.GetBalance(datadir_tmp, address1)
    bal1_2 = _transfers.GetBalance(datadir_tmp, address1_2)
    bal1_3 = _transfers.GetBalance(datadir_tmp, address1_3)
    
    _lib.FatalAssert(bal1 == balances[address1], "Balance is different from group rec for 1")
    _lib.FatalAssert(bal1_2 == balances[address1_2], "Balance is different from group rec for 2")
    _lib.FatalAssert(bal1_3 == balances[address1_3], "Balance is different from group rec for 3")
    
    s1 = bal1 + bal1_2 + bal1_3
    
    startnode.StartNode(datadir_tmp, address1,nodeport)
    datadir = datadir_tmp
    
    #get balaces on nodes wallets
    bal1 = _transfers.GetBalance(datadir, address1)
    bal1_2 = _transfers.GetBalance(datadir, address1_2)
    bal1_3 = _transfers.GetBalance(datadir, address1_3)
    
    s2 = bal1 + bal1_2 + bal1_3
    
    _lib.FatalAssert(s1 == s2, "Balances shoul be equal when a node is On and Off")
    
    #get group balance on a node
    balances = _transfers.GetGroupBalance(datadir)
    _lib.FatalAssert(bal1 == balances[address1], "Balance is different from group rec for 1")
    _lib.FatalAssert(bal1_2 == balances[address1_2], "Balance is different from group rec for 2")
    _lib.FatalAssert(bal1_3 == balances[address1_3], "Balance is different from group rec for 3")
    
    #create 2 wallet locations and 2 wallets in each of them
    walletdatadir1 = _lib.CreateTestFolder("wallet")
    walletdatadir2 = _lib.CreateTestFolder("wallet")
    
    waddress1_1 = _wallet.CreateWallet(walletdatadir1);
    waddress1_2 = _wallet.CreateWallet(walletdatadir1);
    waddress1_3 = _wallet.CreateWallet(walletdatadir1);
    
    waddress2_1 = _wallet.CreateWallet(walletdatadir2);
    waddress2_2 = _wallet.CreateWallet(walletdatadir2);
    
    #send some funds to all that wallets
    amounttosend = "%.8f" % round(bal1[0]/5,8)
    amounttosend3 = "%.8f" % round(bal1_3[0]/5,8)
    
    _transfers.Send(datadir,address1, waddress1_1 ,amounttosend)
    _transfers.Send(datadir,address1, waddress1_2 ,amounttosend)
    _transfers.Send(datadir,address1, waddress2_1 ,amounttosend)
    _transfers.Send(datadir,address1_3, waddress1_3 ,amounttosend3)
    
    # we control how blocks are created. here we wait on a block started and then send another 3 TX
    # we will get 2 more blocks here
    time.sleep(1)
    
    _transfers.Send(datadir,address1, waddress2_2 ,amounttosend)
    amounttosend2 = "%.8f" % round(bal1_2[0]/5,8)
    _transfers.Send(datadir,address1_2, waddress1_1 ,amounttosend2)
    _transfers.Send(datadir,address1_2, waddress1_2 ,amounttosend2)
    
    _transfers.Send(datadir,address1_3, waddress1_3 ,amounttosend3)
    _transfers.Send(datadir,address1_3, waddress1_3 ,amounttosend3)
    
    # wait to complete blocks 
    blocks = _blocks.WaitBlocks(datadir,6)
    time.sleep(2)
    
    _lib.FatalAssert(len(blocks) == 6, "Expected 6 blocks")
    
    #get balances on wallets
    am1 = _wallet.GetBalanceWallet(walletdatadir1, waddress1_1, "localhost", nodeport)
    am2 = _wallet.GetBalanceWallet(walletdatadir1, waddress1_2, "localhost", nodeport)
    am3 = _wallet.GetBalanceWallet(walletdatadir2, waddress2_1, "localhost", nodeport)
    am4 = _wallet.GetBalanceWallet(walletdatadir2, waddress2_2, "localhost", nodeport)
    
    _lib.FatalAssert(am1[1] == round(float(amounttosend) + float(amounttosend2),8), "Expected balance is different for wallet 1_1")
    _lib.FatalAssert(am2[1] == round(float(amounttosend) + float(amounttosend2),8), "Expected balance is different for wallet 1_2")
    _lib.FatalAssert(am3[1] == float(amounttosend), "Expected balance is different for wallet 2_1")
    _lib.FatalAssert(am4[1] == float(amounttosend), "Expected balance is different for wallet 2_2")
    
    #get group blances on a wallet loc
    balances = _transfers.GetGroupBalance(datadir)
    #get balances on node wallets
    
    balances1 = _wallet.GetGroupBalanceWallet(walletdatadir1,"localhost", nodeport)
    balances2 = _wallet.GetGroupBalanceWallet(walletdatadir2,"localhost", nodeport)
    
    _lib.FatalAssert(am1[1] == balances1[waddress1_1][1], "Expected balance is different from group listing for 1_1")
    _lib.FatalAssert(am2[1] == balances1[waddress1_2][1], "Expected balance is different from group listing for 1_2")
    _lib.FatalAssert(am3[1] == balances2[waddress2_1][1], "Expected balance is different from group listing for 2_1")
    _lib.FatalAssert(am4[1] == balances2[waddress2_2][1], "Expected balance is different from group listing for 2_2")
    
    startnode.StopNode(datadir)
    datadir = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()
    


        