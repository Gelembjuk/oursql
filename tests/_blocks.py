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