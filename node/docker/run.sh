#!/bin/bash

# This script starts the database server.
echo "Creating user $DBUSER and database $DBNAME"

#/usr/sbin/mysqld &
/etc/init.d/mysql start

timeout=120
echo -n "Waiting for database server to accept connections"
while ! /usr/bin/mysqladmin -u root status >/dev/null 2>&1
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

if [ -f firststart.lock ]; then

    mysql --default-character-set=utf8 -e "CREATE DATABASE IF NOT EXISTS $DBNAME DEFAULT CHARSET=utf8;"

    echo "Database created"

    mysql --default-character-set=utf8 -e "CREATE USER '$DBUSER' IDENTIFIED BY '$DBPASSWORD'"
    mysql --default-character-set=utf8 -e "GRANT ALL PRIVILEGES ON *.* TO '$DBUSER'@'%' WITH GRANT OPTION; FLUSH PRIVILEGES"
    
    echo "User added"
    
    rm firststart.lock
fi

# Start the second process
./node "$@"

