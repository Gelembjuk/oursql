# OurSQL

![OurSQL](docs/oursql_logo_300.png?raw=true "OurSQL Logo")

Project Web site [http://oursql.org](http://oursql.org)

## Blockchain replication layer for MySQL

OurSQL allows to create a blockchain database quickly with connecting MySQL servers into a cluster.

Use OurSQL to create new blockchain mapped to a MySQL DB or to join to existent blockchain as a cluster node and get MySQL DB data replicated.

### Cryptocurrency

OurSQL supports cryptocurrency too. It is a "side effect" of a blockchain used to replicate data. When blocks are created, some wallet receives coins. It is possible to send conins to any other wallet as it works in bitcoin or similar cryptocurrencies. 

Additionally, it is possible to define "paid" SQL transactions in a [consensus](docs/Consensus.md) rules. To protect some data from unlimited modifications. 

## How  It Works

![How OurSQL Works](docs/oursql_how_it_works.png?raw=true "OurSQL")

OurSQL containes a MySQL proxy server. To make a decetralized application (or "blockchain app") it is needed to code an app that connects to MySQL and works with it like normal single app. Nothing special in the app code. Just standard way to work with MySQL server - update data and select data. 

* The app connects to OurSQL DB port (which is a proxy in fact) and does updates with SQL queries. 
* OurSQL decides if a user can do that SQL query using a consensus rules 
* If it is allowed to do the SQL update quer:
    * OurSQL builds a transaction
    * Adds the transactions to a pool of transactions and executes SQL update
    * Sends a transaction to other known nodes
* If there are enough transactions in a pool , OurSQL makes a block 
* If a node receives a transaction from other node:
    * Checks if that SQL can be executed against a consensus rules
    * Executes and adds to a pool

For the app, all this work is not visible. It just exexutes SQL commands and doesn't care about blockchain or so. OurSQL does all this work itself.

## Consensus

Current version supports only Proof of Work consensus type. Every blockchain has a consensus config file which containes rules. Options of PoW: block hash options, coins to add for minter, numbers of transactions per block etc.

This file is distributed as part of a package instalation package.

Also, in this file it is possible to describe which SQL operation are allowed in this decetralized DB: insert, update, delete , table create. Additionally, it is possible to set rules per table.

Finally, it is possible to set a cost of SQL query per table and type. For example, 0.5 (of a coin) per insert in a table "members". This allows to control updates.

### Future support of consensus 

Soon we add extended support of a consensus management.

We are going to add support of a consensus plug-in. It will be a module for OurSQL (.so or .dll) to control updates.

The consensus module will be able to do some work for every proposed update. Each SQL transaction goes together with a wallet address. The module will be able to check if an address can do this SQL command now. It can do extra requests to other DB table to do some checks, etc.

For example, all users of an app can vote for some user to be a moderator. If a user was elected he is able to update some table. All other are not able. Consensus module can controll such things.

Consensus module will filter all SQL commands received from an app via the proxy and also received from other nodes.

## Try OurSQL

OurSQL is a Golang app. You can compile it yourself or use one of precompiled options.

### Try with Docker

Pull the image

```
docker pull oursql/oursql-server
```

Or just try to run, it will pull automatically.

```
docker run --name oursql -p 8766:8766 -it oursql/oursql-server
```

Now you can connect to the mysql proxy on port 8766 and all updates you do in the DB will be added to a blockchain!

```
mysql -h 127.0.0.1 -P 8766 -u blockchain -pblockchain BC
> CREATE TABLE test (a int unsigned primary key auto_increment, b varchar(100));
> INSERT INTO test SET b='row1';
```

Find more usage [examples](docs/Docker.md) and try to run multiple nodes to see how replication works.

### Compile it yourself

Get OurSQL code files. This works on linux. If you are on other platform the process should be similar.

```
go get github.com/gelembjuk/oursql
```

Install dependencies

```
github.com/go-sql-driver/mysql
go get github.com/JamesStewy/go-mysqldump
go get github.com/btcsuite/btcutil
go get github.com/fatih/structs
go get github.com/mitchellh/mapstructure
```

Go to the node library and build

```
cd $GOPATH/src/github.com/gelembjuk/oursql/node
go build
```

It is ready. Now you can run the OurSQL server. You must have MySQL server running.

You can exacute now 

```
./node
```

Find usage [examples](docs/Tests.md) to do some tests.

### Install

We don't have intallation packages yet. We plan to make it soon. 

### Starting new blockchain DB and consensus management

Blockchain application lifecycle is like:

1. Set up developement environment for your app
    1. Install OurSQL and MySQL
    1. Choose a technology to build your app. It can be anything that can work with MySQL to store data: desktop app or local web server and web app.
1. Prepare a [consensus](docs/Consensus.md) rules
1. Init your blockchain DB. See [examples](docx/Tests.md). When initing, point to your consensus file with the argument -consensusfile PATH_TO_CONSENSUS_CONFIG
1. Build installation package for your application users to install their nodes. The package should inslude:
    1. MySQL server (but you can require it to be installed separately)
    1. OurSQL server
    1. Your application code
    1. Consensus config file
1. *NOTE*. Onse you released your application, you lose control over it! As this is decentralized app. 
    1. If you want to change somethign in the consensus rules, you must rebuild your application installation package and ask users to update. But they must not! Then can say "current consensus is fine".
    1. Someone else can start building your app node code and release it. Blockchain forks are always possible

## Two types of a transactions signing

Each SQL update is a blockchain transaction and it must be signed. There are 2 supported ways to sign a transaction.

1. Signing by a node. If a node config has the option "ProxyKey" set (or sommand line argument -dbproxyaddr), than a node will sign itself all transactions coming from through DB proxy (SQL updates from a client).
1. Signing by a MySQL client. It can be used in case if your node is just a DB proxy and many clients connect to it (meeaning different users with different keys). In this case, SQL updates from a mysql client will require 2 steps to execute.
    * First step - send SQL query and add your public key inside a comment of special format. DB proxy returns a record which contains a data to sign (string to sign)
    * Second step - sign a string with a private key corresponding to public key posted on the first step and do new SQL request where a signature is included as part of a comment of special format.

Read more about [signing of transactions](docs/Signing.md).

## Author

Roman Gelembjuk , roman@gelembjuk.com 

http://gelembjuk.com/

### More info about OurSQL and contacts

[http://oursql.org](http://oursql.org)  

Email: oursql.project@gmail.com

[Twitter: OursqlO](https://twitter.com/OursqlO)

## License

GNU3 