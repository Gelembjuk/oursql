import _lib
import re
import time

def GetBlocks(datadir):
    _lib.StartTest("Load blocks chain")
    res = _lib.ExecuteNode(['printchain','-configdir',datadir, '-view', "short"])
    _lib.FatalAssertSubstr(res,"Hash: ","Blockchain display returned wrong data or no any blocks")
    
    regex = ur"Hash: ([a-z0-9A-Z]+)"

    blocks = re.findall(regex, res)
    
    return blocks

def GetBlocksExt(datadir):
    _lib.StartTest("Load blocks chain. Parse extended")
    res = _lib.ExecuteNode(['printchain','-configdir',datadir, '-view', "short"])
    _lib.FatalAssertSubstr(res,"Hash: ","Blockchain display returned wrong data or no any blocks")
    
    regex = ur"Hash: ([a-z0-9A-Z]+)"

    blocks = re.findall(regex, res)
    
    regex = ur"Transactions: ([0-9]+)"

    transaction = re.findall(regex, res)
    
    blocksr = {}
    
    for ind,b in enumerate(blocks):
        blocksr[b] = transaction[ind]
    
    return blocksr

def WaitBlocks(datadir, explen, maxtime = 10):
    blocks = []
    i = 0
    while True:
        blocks = GetBlocks(datadir)
        
        if len(blocks) >= explen or i >= maxtime:
            break
        time.sleep(1)
        i = i + 1
        
    return blocks

def MintBlock(datadir,minter):
    _lib.StartTest("Force to Mint a block")
    res = _lib.ExecuteNode(['makeblock','-configdir',datadir,'-minter',minter,'-logs','trace'])
    _lib.FatalAssertSubstr(res,"New block mined with the hash","Block making failed")
    
    match = re.search( r'New block mined with the hash ([0-9a-zA-Z]+).', res)

    if not match:
        _lib.Fatal("New block hash can not be found in response "+res)
        
    blockhash = match.group(1)
    
    return blockhash