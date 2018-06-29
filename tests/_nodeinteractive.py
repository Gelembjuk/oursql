import subprocess
import sys
import os
import base64
import json

if len(sys.argv) < 3:
    print "No input info"
    sys.exit(1)

command = json.loads(base64.b64decode(sys.argv[1]))
folder = base64.b64decode(sys.argv[2])

if not isinstance(command, list):
    print "Wrong command"
    sys.exit(2)
 
if not os.path.isdir(folder):
    print "Folder not found"
    sys.exit(3)

f1=open(folder+'/nodestop.txt', 'w+')
 
f1.write("going to fork\n")
 
newpid = os.fork()

if newpid > 0:
    f1.write("child proces id "+str(newpid)+". Exitting main.\n")
    print "Process started with the pid "+str(newpid)
    sys.exit(0)

f1.write("THis is child process. Start command\n")

try:
    res = subprocess.check_output(command)
except subprocess.CalledProcessError as e:
    res = "Returncode: " + e.returncode + ", output:\n" + e.output


f1.write(res)
f1.close()
