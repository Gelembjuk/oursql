# tests all wallet operations with single node

import _lib
import _transfers
import _wallet
import _blocks
import re
import time
import random
import blocksbasic
import startnode
import transactions

datadir = ""

def allowgrouprun():
    return False

def aftertest(testfilter):
    global datadir
    
    if datadir != "":
        startnode.StopNode(datadir)
        
def test(testfilter):
    global datadir
    
    nodeport = '30000'
    
    _lib.CleanTestFolders()
    _lib.StartTestGroup("Init Blockchain")
    
    datadir_tmp = _lib.CreateTestFolder()
    
    r = blocksbasic.PrepareBlockchain(datadir_tmp,nodeport)
    mainaddress = r[0]
    
    startnode.StartNode(datadir_tmp, mainaddress,nodeport)
    datadir = datadir_tmp
    
    walletdatadir = _lib.CreateTestFolder("wallet")
    
    #create 100 wallets
    for i in range(1,101):
        _wallet.CreateWallet(walletdatadir);
        
    addresses = _wallet.GetWallets(walletdatadir)
    
    #send first TX 
    balances = _transfers.GetGroupBalance(datadir)
    
    _transfers.Send(datadir,mainaddress, addresses[0] ,balances[mainaddress][0])
    
    _blocks.WaitBlocks(datadir, 2)
    
    blocks = _blocks.GetBlocksExt(datadir)
    
    _lib.FatalAssert(len(blocks) == 2, "2 blocks are expected")
    
    initialbalance = _wallet.GetBalanceWallet(walletdatadir, addresses[0], "localhost", nodeport)
    
    _lib.FatalAssert(initialbalance[0] == balances[mainaddress][0], "Balance of the first wallet should be same as posted to it")

    _lib.StartTestGroup("Do transactions")

    for i in range(1,100):
        _lib.StartTestGroup("Iteration "+str(i))
        
        addresses = _wallet.GetGroupBalanceWallet(walletdatadir,"localhost",nodeport)
        
        for address in addresses.keys():
            bal = addresses[address][0]
        
            if bal <=0 :
                continue
        
            to = random.choice(addresses.keys())
        
            amount = "%.8f" % round(bal/2,8)
        
            tx = _wallet.Send(walletdatadir,address,to,amount,"localhost",nodeport)
            
        blocks = _blocks.GetBlocksExt(datadir)
        print "\t\tNumer of blocks %d"%(len(blocks))

        
    startnode.StopNode(datadir)
    datadir = ""
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()