#!/bin/bash

ps aux |grep mysqld |grep -q -v grep
PROCESS_1_STATUS=$?

ps aux |grep node |grep -q -v grep
PROCESS_2_STATUS=$?
# If the greps above find anything, they exit with 0 status
# If they are not both 0, then something is wrong
if [ $PROCESS_1_STATUS -ne 0 -o $PROCESS_2_STATUS -ne 0 ]; then
echo "One of the processes has already exited."
exit 1
fi

exit 0