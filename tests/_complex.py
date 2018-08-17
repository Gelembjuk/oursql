from shutil import copyfile

import _lib
import _transfers
import _blocks
import re
import os
import os.path
import time
import json
import random
import startnode
import blocksnodes
import blocksbasic
import managenodes
import transactions

def PrepareNodes():
    
    nodeport = '30000'
    
    _lib.StartTestGroup("Wallet Balance")
    
    _lib.CleanTestFolders()
    
    datadir_tmp = CopyBlockchainWithBlocks("_1_")
    
    blocks = _blocks.GetBlocks(datadir_tmp)
    
    balances = _transfers.GetGroupBalance(datadir_tmp)
    
    datadirs = []
    
    address1 = balances.keys()[0]
    
    # address1_3 becomes a minter. we will send money from other 2 and this will receive rewards
    startnode.StartNode(datadir_tmp, address1,nodeport)
    datadir = datadir_tmp
    datadirs.append(datadir_tmp)
    
    nodes = []
    
    for i in range(1, 6):
        port = str(30000+i)
        d = blocksnodes.StartNodeAndImport(port, nodeport, "Server "+str(i),"_"+str(i+1)+"_")
        datadir_n = d[0]
        address_n = d[1]
        
        nodes.append({'number':i, 'port':port, 'datadir':datadir_n,'address':address_n})
        datadirs.append(datadir_n)
    
    _lib.StartTestGroup("Temp Data Dirs")
    _lib.StartTest("Node 0 "+os.path.basename(datadir))
    
    for node in nodes:
        _lib.StartTest("Node "+str(node['number'])+" "+os.path.basename(node['datadir']))
        
    # commmon transfer of blocks between nodes
    _lib.StartTestGroup("Transfer of blocks between nodes")
    
    blocks = _blocks.GetBlocks(datadir)
    blockslen = len(blocks)
    
    balance1 = _transfers.GetBalance(datadir, address1)
    as1 = "%.8f" % round(balance1[0]/10,8)
    
    # should be 6 transactions
    _transfers.Send(datadir,address1, nodes[0]['address'] ,as1)
    _transfers.Send(datadir,address1, nodes[1]['address'] ,as1)
    _transfers.Send(datadir,address1, nodes[2]['address'] ,as1)
    
    _transfers.Send(datadir,address1, nodes[0]['address'] ,as1)
    _transfers.Send(datadir,address1, nodes[1]['address'] ,as1)
    _transfers.Send(datadir,address1, nodes[2]['address'] ,as1)
    
    blocks = _blocks.WaitBlocks(datadir, blockslen + 1)
    
    _lib.FatalAssert(len(blocks) == blockslen + 1, "Expected "+str(blockslen +1)+" blocks")
    
    #wait while block is posted to all other nodes
    time.sleep(1)
    # check on each node
    for node in nodes:
        blocks = _blocks.WaitBlocks(node['datadir'], blockslen + 1)
        _lib.FatalAssert(len(blocks) == blockslen + 1, "Expected "+str(blockslen +1)+" blocks o node "+str(node['number']))
    
    _lib.StartTestGroup("Create 2 branches of blockchain")
    
    # remove connection between subnetworks
    managenodes.RemoveAllNodes(nodes[0]['datadir'])
    managenodes.RemoveAllNodes(nodes[1]['datadir'])
    managenodes.RemoveAllNodes(nodes[2]['datadir'])
    managenodes.RemoveAllNodes(nodes[3]['datadir'])
    managenodes.RemoveAllNodes(nodes[4]['datadir'])
    managenodes.RemoveAllNodes(datadir)
    
    # first group - main and 4,5 nodes
    managenodes.AddNode(datadir,"localhost",'30004')
    managenodes.AddNode(datadir,"localhost",'30005')
    managenodes.AddNode(nodes[3]['datadir'],"localhost",'30005')
    
    #second group 1,2,3
    managenodes.AddNode(nodes[0]['datadir'],"localhost",'30002')
    managenodes.AddNode(nodes[0]['datadir'],"localhost",'30003')
    managenodes.AddNode(nodes[1]['datadir'],"localhost",'30003')
    
    time.sleep(1)
    
    #check nodes
    
    nodes0 = managenodes.GetNodes(datadir)
    _lib.FatalAssert("localhost:30005", "Node 5 is not in the list of 0")
    _lib.FatalAssert("localhost:30004", "Node 4 is not in the list of 0")
    
    nodes1 = managenodes.GetNodes(nodes[0]['datadir'])
    
    _lib.FatalAssert("localhost:30002", "Node 2 is not in the list of 1")
    _lib.FatalAssert("localhost:30003", "Node 3 is not in the list of 1")
    
    nodes2 = managenodes.GetNodes(nodes[1]['datadir'])
    
    _lib.FatalAssert("localhost:30001", "Node 1 is not in the list of 2")
    _lib.FatalAssert("localhost:30003", "Node 3 is not in the list of 2")
    
    _lib.StartTestGroup("2 new blocks on first branch")
    
    balance1 = _transfers.GetBalance(datadir, address1)
    as1 = "%.8f" % round(balance1[0]/20,8)
    
    # 7 TX for 8-th block
    tx = [""] * 15
    tx[0] = _transfers.Send(datadir,address1, nodes[3]['address'] ,as1)
    print tx[0]
    tx[1] = _transfers.Send(datadir,address1, nodes[3]['address'] ,as1)
    print tx[1]
    tx[2] = _transfers.Send(datadir,address1, nodes[3]['address'] ,as1)
    print tx[2]
    tx[3] = _transfers.Send(datadir,address1, nodes[3]['address'] ,as1)
    print tx[3]
    tx[4] = _transfers.Send(datadir,address1, nodes[3]['address'] ,as1)
    print tx[4]
    tx[5] = _transfers.Send(datadir,address1, nodes[3]['address'] ,as1)
    print tx[5]
    tx[6] = _transfers.Send(datadir,address1, nodes[3]['address'] ,as1)
    print tx[6]
    
    time.sleep(7)
    # 8 TX for 9-th block
    tx[7] = _transfers.Send(datadir,address1, nodes[4]['address'] ,as1)
    print tx[7]
    tx[8] = _transfers.Send(datadir,address1, nodes[4]['address'] ,as1)
    print tx[8]
    tx[9] = _transfers.Send(datadir,address1, nodes[4]['address'] ,as1)
    print tx[9]
    tx[10] = _transfers.Send(datadir,address1, nodes[4]['address'] ,as1)
    print tx[10]
    tx[11] = _transfers.Send(datadir,address1, nodes[4]['address'] ,as1)
    print tx[11]
    tx[12] = _transfers.Send(datadir,address1, nodes[4]['address'] ,as1)
    print tx[12]
    tx[13] = _transfers.Send(datadir,address1, nodes[4]['address'] ,as1)
    print tx[13]
    tx[14] = _transfers.Send(datadir,address1, nodes[4]['address'] ,as1)
    print tx[14]
    
    blocks1 = _blocks.WaitBlocks(nodes[4]['datadir'], blockslen + 3)
    time.sleep(3)
    
    _lib.FatalAssert(len(blocks1) == blockslen + 3, "Expected "+str(blockslen +3)+" blocks for branch 1")
    
    _lib.StartTestGroup("1 new block on second branch")
    
    balance2 = _transfers.GetBalance(nodes[0]['datadir'], nodes[0]['address'])
    as2 = "%.8f" % round(balance2[0]/10,8)
    
    # 7 new TX
    tx1 = _transfers.Send(nodes[0]['datadir'], nodes[0]['address'], nodes[1]['address'] ,as2)
    tx2 = _transfers.Send(nodes[0]['datadir'], nodes[0]['address'], nodes[2]['address'] ,as2)
    tx3 = _transfers.Send(nodes[0]['datadir'], nodes[0]['address'], nodes[2]['address'] ,as2)
    _transfers.Send(nodes[0]['datadir'], nodes[0]['address'], nodes[2]['address'] ,as2)
    _transfers.Send(nodes[0]['datadir'], nodes[0]['address'], nodes[2]['address'] ,as2)
    _transfers.Send(nodes[0]['datadir'], nodes[0]['address'], nodes[2]['address'] ,as2)
    _transfers.Send(nodes[0]['datadir'], nodes[0]['address'], nodes[2]['address'] ,as2)
    
    blocks2 = _blocks.WaitBlocks(nodes[2]['datadir'], blockslen + 2)
    _lib.FatalAssert(len(blocks2) == blockslen + 2, "Expected "+str(blockslen +2)+" blocks for branch 2")
    
    dstdir = _lib.getCurrentDir()+"/datafortests/"
    #configs for cluster 1
    copyConfig(datadir, dstdir+"bc6nodes_1/config.t", address1, nodeport, nodes[3]['port'], nodes[4]['port'])
    
    copyConfig(nodes[3]['datadir'], dstdir+"bc6nodes_5/config.t", nodes[3]['address'], nodes[3]['port'], nodeport, nodes[4]['port'])
    
    copyConfig(nodes[4]['datadir'], dstdir+"bc6nodes_6/config.t", nodes[4]['address'], nodes[4]['port'], nodeport, nodes[3]['port'])
    
    #config for cluster 2
    copyConfig(nodes[0]['datadir'], dstdir+"bc6nodes_2/config.t", nodes[0]['address'], nodes[0]['port'], nodes[1]['port'], nodes[2]['port'])
    
    copyConfig(nodes[1]['datadir'], dstdir+"bc6nodes_3/config.t", nodes[1]['address'], nodes[1]['port'], nodes[0]['port'], nodes[2]['port'])
    
    copyConfig(nodes[2]['datadir'], dstdir+"bc6nodes_4/config.t", nodes[2]['address'], nodes[2]['port'], nodes[0]['port'], nodes[1]['port'])
    
    #print os.path.basename(datadir)
    startnode.StopNode(datadir)
    
    dstdir = _lib.getCurrentDir()+"/datafortests/bc6nodes_1/"
    #print dstdir

    copyfile(datadir+"/wallet.dat", dstdir+"wallet.t")
    
    try:
        os.remove(dstdir+"db.sql")
    except OSError:
        pass

    DumpBCDB(datadir, dstdir+"db.sql")
    
    i = 2
    
    for node in nodes:
        #print os.path.basename(node['datadir'])
        startnode.StopNode(node['datadir'])
        
        dstdir = _lib.getCurrentDir()+"/datafortests/bc6nodes_"+str(i)+"/"
        #print dstdir
        copyfile(node['datadir']+"/wallet.dat", dstdir+"wallet.t")
        
        try:
            os.remove(dstdir+"db.sql")
        except OSError:
            pass
        
        DumpBCDB(node['datadir'], dstdir+"db.sql")
        
        i=i+1
    
    return [datadir, address1, nodes, blockslen+1, datadirs]

def copyConfig(datadir, destfile,minter, port, port2, port3):
    origjsonfile = datadir+"/config.json"
    with open(origjsonfile) as f:
        data = json.load(f)
        f.close()

        data["Minter"] = minter
        data["Port"] = int(port)
        if not "Nodes" in data:
            data["Nodes"] = []
        if len(data["Nodes"]) == 0:
            data["Nodes"].append({})
        data["Nodes"][0] = {"Host":"localhost", "Port": int(port2)}
        if len(data["Nodes"]) == 1:
            data["Nodes"].append({})
        data["Nodes"][1] = {"Host":"localhost", "Port": int(port3)} 
        
        data["Database"]["DatabaseName"] = ""
        
        with open(destfile, 'w') as fp:
            json.dump(data, fp)
    
 
def DumpBCDB(datadir, dumpfile):
    if os.path.isfile(dumpfile):
        os.remove(dumpfile)
        
    _lib.StartTest("Dump BC DB for future tests")
    res = _lib.ExecuteNode(['dumpblockchain','-configdir',datadir,'-dumpfile',dumpfile])
    _lib.FatalAssertSubstr(res,"Blockchain DB was dumped to a file","Dump was not succes")

def AddMinterToConfig(datadir, minter):
    res = _lib.ExecuteNode(['updateconfig','-configdir',datadir,'-minter',minter])
 
def CopyBlockchainWithBlocks(suffix = ""):
    datadir = _lib.CreateTestFolder(suffix)
    _lib.CopyTestData(datadir,"bcwith4blocks")
    
    return datadir

def Copy6Nodes():
    datadirs = [""] * 6
    datadirs[0] = _lib.CreateTestFolder("_1_")
    _lib.CopyTestData(datadirs[0],"bc6nodes_1")
    
    datadirs[1] = _lib.CreateTestFolder("_2_")
    _lib.CopyTestData(datadirs[1],"bc6nodes_2")
    
    datadirs[2] = _lib.CreateTestFolder("_3_")
    _lib.CopyTestData(datadirs[2],"bc6nodes_3")
    
    datadirs[3] = _lib.CreateTestFolder("_4_")
    _lib.CopyTestData(datadirs[3],"bc6nodes_4")
    
    datadirs[4] = _lib.CreateTestFolder("_5_")
    _lib.CopyTestData(datadirs[4],"bc6nodes_5")
    
    datadirs[5] = _lib.CreateTestFolder("_6_")
    _lib.CopyTestData(datadirs[5],"bc6nodes_6")
    
    return datadirs

def Make5BlocksBC():
    datadir = _lib.CreateTestFolder()
    
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
    
    # create another 3 addresses
    address2 = transactions.CreateWallet(datadir)
    address3 = transactions.CreateWallet(datadir)

    _lib.StartTestGroup("Do transfers")

    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    amount1 = '1'
    amount2 = '2'
    amount3 = '3'
    
    _lib.StartTestGroup("Send 1 transaction")
    
    # one TX in first block
    txid1 = _transfers.Send(datadir,address,address2,amount1)
    txlist = transactions.GetUnapprovedTransactions(datadir)
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
    
    blockchash = _blocks.MintBlock(datadir,address)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 2,"Should be 2 blocks in blockchain")
    
    _lib.StartTestGroup("Send 2 transactions")
    # 2 TX in second block
    txid2 = _transfers.Send(datadir,address,address3,amount2)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    _lib.FatalAssert(len(txlist) == 1,"Should be 1 unapproved transaction")
    
    txid3 = _transfers.Send(datadir,address,address3,amount3)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 2,"Should be 2 unapproved transaction")
    
    blockchash = _blocks.MintBlock(datadir,address)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 3,"Should be 3 blocks in blockchain")
   
    _lib.StartTestGroup("Send 3 transactions")
    
    microamount = 0.01
    
    txid1 = _transfers.Send(datadir,address,address2,microamount)
    txid2 = _transfers.Send(datadir,address2,address3,microamount)
    txid3 = _transfers.Send(datadir,address3,address,microamount)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 3,"Should be 3 unapproved transaction")
    
    blockchash = _blocks.MintBlock(datadir,address)
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 4,"Should be 4 blocks in blockchain")
    
    _lib.StartTestGroup("Send 4 transactions. Random value")
    
    microamountmax = 0.01
    microamountmin = 0.0095
    
    a1 = round(random.uniform(microamountmin, microamountmax),8)
    a2 = round(random.uniform(microamountmin, microamountmax),8)
    a3 = round(random.uniform(microamountmin, microamountmax),8)
    a4 = round(random.uniform(microamountmin, microamountmax),8)
    txid1 = _transfers.Send(datadir,address,address2,a1)
    txid2 = _transfers.Send(datadir,address2,address3,a2)
    txid3 = _transfers.Send(datadir,address3,address,a3)
    txid3 = _transfers.Send(datadir,address3,address,a4)
    
    txlist = transactions.GetUnapprovedTransactions(datadir)
    
    _lib.FatalAssert(len(txlist) == 4,"Should be 4 unapproved transaction")
    
    if txid1 not in txlist.keys():
        _lib.Fatal("Transaction 1 is not in the list of transactions after iteration "+str(i))

    if txid2 not in txlist.keys():
        _lib.Fatal("Transaction 2 is not in the list of transactions after iteration "+str(i))

    if txid3 not in txlist.keys():
        _lib.Fatal("Transaction 3 is not in the list of transactions after iteration "+str(i))
        
    blockchash = _blocks.MintBlock(datadir,address)
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 5,"Should be 5 blocks in blockchain")
    
    _lib.StartTestGroup("Send 5 transactions. Random value")
    
    txid1 = _transfers.Send(datadir,address,address2,a4)
    txid2 = _transfers.Send(datadir,address2,address3,a3)
    txid3 = _transfers.Send(datadir,address3,address,a2)
    txid3 = _transfers.Send(datadir,address3,address,a1)
    txid1 = _transfers.Send(datadir,address,address2,a1)
    
    blockchash = _blocks.MintBlock(datadir,address)
    transactions.GetUnapprovedTransactionsEmpty(datadir)
    
    blockshashes = _blocks.GetBlocks(datadir)
    
    _lib.FatalAssert(len(blockshashes) == 6,"Should be 6 blocks in blockchain")
    
    dstdir = _lib.getCurrentDir()+"/datafortests/bcwith4blocks/"
    
    copyfile(datadir+"/wallet.dat", dstdir+"wallet.t")
    DumpBCDB(datadir, dstdir+"db.sql")
    #copyfile(datadir+"/nodeslist.db", dstdir+"nodeslist.t")
    #copyfile(datadir+"/blockchain.db", dstdir+"blockchain.t")
    
    return [datadir, address, address2, address3]