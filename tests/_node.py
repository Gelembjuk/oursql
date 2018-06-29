import _lib
import re
import time

def StartNodeInteractive(datadir, address, port,comment = ""):
    _lib.StartTest("Start node (debug) "+comment)
    res = _lib.ExecuteHangNode(['startintnode','-configdir',datadir,'-port',port,'-minter',address],datadir)
    _lib.FatalAssertSubstr(res,"Process started","No process start marker")

def GetWallets(datadir):
    _lib.StartTest("Get node wallets")
    res = _lib.ExecuteNode(['listaddresses','-configdir',datadir])
    
    _lib.FatalAssertSubstr(res,"Wallets (addresses)","No list of wallets")
    
    regex = ur"(1[a-zA-Z0-9]{30,100})"

    addresses = re.findall(regex, res)
    
    return addresses

def NodeState(datadir):
    _lib.StartTest("Check node state")
    res = _lib.ExecuteNode(['nodestate','-configdir',datadir])
    _lib.FatalAssertSubstr(res,"Number of blocks","No info about blocks")
    
    state = {}
    
    match = re.search( r'Number of blocks - (\d+)', res)

    if not match:
        _lib.Fatal("Number of blocks is not found "+res)
        
    state['blocks'] = match.group(1)
    
    match = re.search( r'Number of unapproved transactions - (\d+)', res)

    if not match:
        _lib.Fatal("Numberof unapproved transactions not found "+res)
        
    state['unapproved'] = match.group(1)
    
    match = re.search( r'Number of unspent transactions outputs - (\d+)', res)

    if not match:
        _lib.Fatal("Number of unspent transactions outputs -  not found "+res)
        
    state['unspent'] = match.group(1)
    
    state['inprogress'] = False
    
    match = re.search( r'Loaded (\d+) of (\d+) blocks', res)

    if match:
        state['totalnumber'] = match.group(2)
        state['inprogress'] = True
    
    return state

def WaitBlocksInState(datadir, explen, maxtime = 10):
    i = 0
    while True:
        state = NodeState(datadir)
        
        if int(state['blocks']) >= explen or i >= maxtime:
            break
        time.sleep(1)
        i = i + 1
        
    return state