#!/bin/bash

# Start mysql process
mysqld &

timeout=120
echo -n "Waiting for database server to accept connections"
while ! mysql -u root -e "SHOW DATABASES"
do
    timeout=$(($timeout - 1))
    if [ $timeout -eq 0 ]; then
        echo -e "\nCould not connect to database server. Aborting..."
        exit 1
    fi
    
    echo -n "."
    sleep 1
done
echo
# wait 2 seconds while mysql finally started

# Start the second process
./node "$@"

while sleep 60; do
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
done