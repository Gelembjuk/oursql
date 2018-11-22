import _lib
import _transfers
import _blocks
import _complex
import _node
import re
import os
import time
import random
import startnode
import blocksnodes
import managenodes
import transactions

datadirs = []

def aftertest(testfilter):
    global datadirs
    
    for datadir in datadirs:
        if datadir != "":
            startnode.StopNode(datadir)
        
def test(testfilter):
    global datadirs
    _lib.CleanTestFolders()
    
    
    #_lib.RemoveTestFolder(datadir)
    _lib.EndTestGroupSuccess()


