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

# remove pid file if it exists. It should not exist but can be if previous stop was not really clean
if [ -f conf/server.pid ]; then
    rm conf/server.pid
fi

# update hosts file to have special localhost hostname to connect to HOST as to localhost
sed '/host.local.address/d' /etc/hosts > /etc/hosts.tmp
cat /etc/hosts.tmp > /etc/hosts
rm /etc/hosts.tmp
#detect IP of a host

/sbin/ip route
/sbin/ip route|awk '/default/ { print $3 }'

HOSTIP=$(/sbin/ip route|awk '/default/ { print $3 }')
echo "host IP is $HOSTIP . Updating hosts file"
echo "$HOSTIP host.local.address" >> /etc/hosts

function finish {
    # stop started subprocesses
    kill -15 node
    /etc/init.d/mysql stop
}
trap finish SIGINT SIGTERM

# Start the second process
./node "$@"

