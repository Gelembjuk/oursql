import sys
import re
import _lib
from os import listdir
from os.path import isfile, join
import traceback

test = ""

if len(sys.argv) > 1 :
    test = sys.argv[1]
    m = re.search(r'^([a-z].+)\.py$',test)
    if m:
        test = m.group(1)

if test == "":
    test = "all"
    
# read all test files from this dir 
curdir = _lib.getCurrentDir()

testfiles = [f for f in listdir(curdir) if isfile(join(curdir, f)) and re.search(r'^[a-z].+\.py$',f)]

tests = []

for testscript in testfiles:
    if test == "all" or test+'.py' == testscript:
        m = re.search(r'^([a-z].+)\.py$',testscript)
        
        if not m:
            continue
        
        testname = m.group(1)
        
        tests.append(testname)

num = 1
failed = []
passed = []

for testname in tests:
    if test == "all" or test == testname:
        if test == "all":
            print "######## ",testname, str(num)+" of "+str(len(tests))
        
        test_module = __import__(testname)
        
        methods = dir(test_module)

        if "allowgrouprun" in methods and test == "all":
            if not test_module.allowgrouprun():
                continue
            

        if "beforetest" in methods:
            test_module.beforetest(testname)
        
        success = True
        
        try:
            test_module.test(testname)
        except NameError as e:
            print "Name exception: ", e
            success = False
        except:
            # do nothing. We catch exception only to be able to execute end function
            e = sys.exc_info()[1]
            print "Error: ",e
            traceback.print_exc(file=sys.stdout)
            success = False
            
        if "aftertest" in methods:
            test_module.aftertest(testname)
            
        if not success:
            break
        
        passed.append(testname)
        num = num + 1

if test == "all":
    passedlen = len(passed)
    print str(passedlen)+" of "+str(len(tests))+" tests passed"