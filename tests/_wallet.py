import _lib
import re

def parseWalletBalance(res, address):
    # get balance from this response 
    match = re.search( r'Balance of \'([^\']+)\':', res)

    if not match:
        _lib.Fatal("Address can not be found in "+res)
    
    addr = match.group(1)
    
    _lib.FatalAssert(addr == address, "Address in a response is not same as requested. "+res)
    
    balance = [0,0,0];
    
    match = re.search( r'Approved\s+-\s+([0-9.]+)', res)

    if not match:
        _lib.Fatal("Approved Balance can not be found in "+res)
    
    balance[1] = round(float(match.group(1)),8)
    
    match = re.search( r'Total\s+-\s+([0-9.]+)', res)

    if not match:
        _lib.Fatal("Total Balance can not be found in "+res)
    
    balance[0] = round(float(match.group(1)),8)
    
    match = re.search( r'Pending\s+-\s+([0-9.-]+)', res)

    if not match:
        _lib.Fatal("Pending Balance can not be found in "+res)
    
    balance[2] = round(float(match.group(1)),8)
    
    return balance

def GetBalanceWallet(datadir, address, host, port):
    _lib.StartTest("Request balance for a wallet "+address)
    res = _lib.ExecuteWallet(['getbalance','-configdir',datadir,"-address",address,"-nodehost",host,"-nodeport",port])
    _lib.FatalAssertSubstr(res,"Balance of","Balance info is not found")

    return parseWalletBalance(res, address)

def GetBalanceWalletNoNode(datadir, address):
    _lib.StartTest("Request balance for a wallet "+address)
    res = _lib.ExecuteWallet(['getbalance','-configdir',datadir,"-address",address])
    _lib.FatalAssertSubstr(res,"Balance of","Balance info is not found")

    return parseWalletBalance(res, address)

def GetGroupBalanceWallet(datadir,host,port):
    _lib.StartTest("Request group balance for addresses in a wallet")
    res = _lib.ExecuteWallet(['listbalances','-configdir',datadir,"-nodehost",host,"-nodeport",port])
    _lib.FatalAssertSubstr(res,"Balance for all addresses:","Balance result not printed")

    regex = ur"([a-z0-9A-Z]+): ([0-9.]+) .Approved - ([0-9.]+), Pending - ([0-9.-]+)"

    balancesres = re.findall(regex, res)
    balances = {}
    
    for r in balancesres:
        balances[r[0]] = [round(float(r[1]),8),round(float(r[2]),8),round(float(r[3]),8)]
    
    return balances

def GetGroupBalanceWalletNoNode(datadir):
    _lib.StartTest("Request group balance for addresses in a wallet")
    res = _lib.ExecuteWallet(['listbalances','-configdir',datadir])
    _lib.FatalAssertSubstr(res,"Balance for all addresses:","Balance result not printed")

    regex = ur"([a-z0-9A-Z]+): ([0-9.]+) .Approved - ([0-9.]+), Pending - ([0-9.-]+)"

    balancesres = re.findall(regex, res)
    balances = {}
    
    for r in balancesres:
        balances[r[0]] = [round(float(r[1]),8),round(float(r[2]),8),round(float(r[3]),8)]
    
    return balances

def CreateWallet(datadir):
    _lib.StartTest("Create new wallet")
    res = _lib.ExecuteWallet(['createwallet','-configdir',datadir])
    _lib.FatalAssertSubstr(res,"Your new address","Address creation failed")
    match = re.search( r'.+: (.+)', res)

    if not match:
        _lib.Fatal("Address can not be found in "+res)
        
    address = match.group(1)
    
    return address

def GetWallets(datadir):
    _lib.StartTest("Get wallets")
    res = _lib.ExecuteWallet(['listaddresses','-configdir',datadir])
    _lib.FatalAssertSubstr(res,"Wallets (addresses)","No list of wallets")
    
    regex = ur"(1[a-zA-Z0-9]{30,100})"

    addresses = re.findall(regex, res)
    
    return addresses

def Send(datadir,fromaddr,to,amount,host,port):
    _lib.StartTest("Send money. From "+fromaddr+" to "+to+" amount "+str(amount))
    
    res = _lib.ExecuteWallet(['send','-configdir',datadir,'-from',fromaddr,'-to',to,'-amount',str(amount),"-nodehost",host,"-nodeport",port])
    
    _lib.FatalAssertSubstr(res,"Success. New transaction:","Sending of money failed. NO info about new transaction")
    
    # get transaction from this response 
    match = re.search( r'Success. New transaction: (.+)', res)

    if not match:
        _lib.Fatal("Transaction ID can not be found in "+res)
        
    txid = match.group(1)

    return txid

def SendNoNode(datadir,fromaddr,to,amount):
    _lib.StartTest("Send money. From "+fromaddr+" to "+to+" amount "+str(amount))
    
    res = _lib.ExecuteWallet(['send','-configdir',datadir,'-from',fromaddr,'-to',to,'-amount',str(amount)])
    
    _lib.FatalAssertSubstr(res,"Success. New transaction:","Sending of money failed. NO info about new transaction")
    
    # get transaction from this response 
    match = re.search( r'Success. New transaction: (.+)', res)

    if not match:
        _lib.Fatal("Transaction ID can not be found in "+res)
        
    txid = match.group(1)

    return txid

def SendTooMuch(datadir,fromaddr,to,amount,host,port):
    _lib.StartTest("Send too much money. From "+fromaddr+" to "+to+" amount "+str(amount))
    res = _lib.ExecuteWallet(['send','-configdir',datadir,'-from',fromaddr,'-to',to,'-amount',str(amount),"-nodehost",host,"-nodeport",port])
    _lib.FatalAssertSubstr(res,"No enough funds","Sending of money didn't fail as expected")
    
def SendTooMuchNoNode(datadir,fromaddr,to,amount):
    _lib.StartTest("Send too much money. From "+fromaddr+" to "+to+" amount "+str(amount))
    res = _lib.ExecuteWallet(['send','-configdir',datadir,'-from',fromaddr,'-to',to,'-amount',str(amount)])
    _lib.FatalAssertSubstr(res,"No enough funds","Sending of money didn't fail as expected")
    
def SetNodeConfig(datadir,host,port):
    _lib.StartTest("Set config file for wallet")
    res = _lib.ExecuteWallet(['setnode','-configdir',datadir,'-nodehost',host,'-nodeport',port])
    _lib.FatalAssertSubstr(res,"Config updated","Config update failed")
    
def GetUnspentNoNode(datadir,address):
    _lib.StartTest("Get unspent transactions list")
    res = _lib.ExecuteWallet(['showunspent','-configdir',datadir,'-address',address])
    _lib.FatalAssertSubstr(res,"Balance - ","No list of transactions and balance")
    
    regex = ur"([0-9.]+)\s+from\s+(.+) in transaction (.+) output #(\d+)"

    txres = re.findall(regex, res)
    
    return txres

def GetUnspent(datadir,address,host, port):
    _lib.StartTest("Get unspent transactions list")
    res = _lib.ExecuteWallet(['showunspent','-configdir',datadir,'-address',address,"-nodehost",host,"-nodeport",port])
    _lib.FatalAssertSubstr(res,"Balance - ","No list of transactions and balance")
    
    regex = ur"([0-9.]+)\s+from\s+(.+) in transaction (.+) output #(\d+)"

    txres = re.findall(regex, res)
    
    return txres 

def GetHistoryNoNode(datadir,address):
    _lib.StartTest("Get address transactions history for "+address)
    res = _lib.ExecuteWallet(['showhistory','-configdir',datadir,'-address',address])
    _lib.FatalAssertSubstr(res,"History of transactions","No history result")
    
    regex = ur"([0-9.]+)\s+(Out To|In from)\s+([a-zA-Z0-9]+)"

    hist = re.findall(regex, res)
    
    return hist

def GetHistory(datadir,address,host, port):
    _lib.StartTest("Get address transactions history for "+address)
    res = _lib.ExecuteWallet(['showhistory','-configdir',datadir,'-address',address,"-nodehost",host,"-nodeport",port])
    _lib.FatalAssertSubstr(res,"History of transactions","No history result")
    
    regex = ur"([0-9.]+)\s+(Out To|In from)\s+([a-zA-Z0-9]+)"

    hist = re.findall(regex, res)
    
    return hist