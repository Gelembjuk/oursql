import _lib
import re
import time

def ExecuteSQL(datadir,fromaddr,sqlcommand):
    _lib.StartTest("Execute SQL by "+fromaddr+" "+sqlcommand)

    res = _lib.ExecuteNode(['sql','-configdir',datadir,'-from',fromaddr,'-sql',sqlcommand])
    
    _lib.FatalAssertSubstr(res,"Success. New transaction:","Executing SQL failes. NO info about new transaction")
    
    # get transaction from this response 
    match = re.search( r'Success. New transaction: (.+)', res)

    if not match:
        _lib.Fatal("Transaction ID can not be found in "+res)
        
    txid = match.group(1)

    return txid

    