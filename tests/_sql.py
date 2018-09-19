import _lib
import re
import time

def ExecuteSQL(datadir,fromaddr,sqlcommand):
    _lib.StartTest("Execute SQL by "+fromaddr+" "+sqlcommand)

    res = _lib.ExecuteNode(['sql','-configdir',datadir,'-from',fromaddr,'-sql',sqlcommand])
    
    _lib.FatalAssertSubstr(res,"Success. New transaction:","Executing SQL failes. NO info about new transaction. SQL error")
    
    # get transaction from this response 
    match = re.search( r'Success. New transaction: (.+)', res)

    if not match:
        _lib.Fatal("Transaction ID can not be found in "+res)
        
    txid = match.group(1)

    return txid

def ExecuteSQLFailure(datadir,fromaddr,sqlcommand):
    _lib.StartTest("Execute SQL by "+fromaddr+" "+sqlcommand+" , expect failure")

    res = _lib.ExecuteNode(['sql','-configdir',datadir,'-from',fromaddr,'-sql',sqlcommand])
    
    _lib.FatalAssertSubstr(res,"Error: ","Error was expected")
    
    # get transaction from this response 
    match = re.search( r'Error: (.+)', res)

    if not match:
        _lib.Fatal("No error message")
        
    error = match.group(1)

    return error

def ExecuteSQLOnProxy(datadir,sqlcommand):
    _lib.StartTest("Execute SQL on Proxy "+sqlcommand)

    res = _lib.DBExecute(datadir,sqlcommand,True)
    
    _lib.FatalAssert(res=="","Error for proxy SQL call: "+res)

    return True

def ExecuteSQLOnProxyFail(datadir,sqlcommand):
    _lib.StartTest("Execute SQL on Proxy "+sqlcommand)

    res = _lib.DBExecute(datadir,sqlcommand,True)
    
    _lib.FatalAssert(res!="","Error was expected. But query is success")

    return True