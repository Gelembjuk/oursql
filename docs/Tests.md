# OurSQL Demo tests

## Compile OurSQL

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

## Tests

Run the app. It will display the help prompt

```
./node
```

First action is to create a wallet - a pair of cryptographic keys, private and public keys.

```
./node createwallet
# to list all your wallets do
./node listaddresses
```

Init your new blockchain. You need empty DB on MySQL server for this. We will use the DB named BC and mysql user blockchain/blockchain. YOUR_WALLET_ADDRESS is your wallet address (output of createwallet command). This is the wallet that will receive cooins when this node mints new block of blockchain.

```
./node initblockchain -minter YOUR_WALLET_ADDRESS  -mysqluser blockchain -mysqlpass blockchain -mysqldb BC 
```

Ths command creates the first block of your blockchain. Next action is to update your nods configs to prevent command line arguments every time.

You need to set your wallet address again, TCP port where your node communicates with other nodes and your DB proxy port to communicate with mysql clients. proxykey defines a wallet that will sign all SQL transactions processed by the DB proxy. It can be different address from "minter". Also, it can be missed, in this case a client has to sign transactions itself (this documentation is coming).

```
./node updateconfig -minter YOUR_WALLET_ADDRESS -port 8765 -dbproxyaddr :8766 -proxykey YOUR_WALLET_ADDRESS
```

Now you can start your node

```
./node startnode
# if you want to stop do
./node stopnode
# to check node state do
./node nodestate
```

And all is set to connect with mysql client. Use standard command line mysql client (or any other you prefer). You are connecting to the proxy server, not directly to the DB server.

```
mysql -h 127.0.0.1 -P 8766 -u blockchain -pblockchain BC
# Create your table synced with blockchain
> CREATE TABLE test (a INT UNSIGNED PRIMARY KEY auto_increment, b VARCHAR(100));
> INSERT INTO test SET b='row1';
```

Now see the state of the node
```
./node nodestate
# print blockchain info
./node printchain
# print transactions pool
./node unapprovedtransactions
```

There is one transaction in a pool. Because minimum transactions per block must be not less than a block height. For 2-nd block - 1 transaction (create table was anough), for 3-rd block we need 2 transactions (+ coin base) and so on.

Do one more SQL update to get one more transaction and your new block will be made.

```
mysql -h 127.0.0.1 -P 8766 -u blockchain -pblockchain BC
> INSERT INTO test SET b='row2';
```

And check list of blocks. Note, OurSQL needs some time to make a block. By default it is small time, 3 seconds or so. By default , there is used weak Proof Of Work rule.

```
./node printchain
./node unapprovedtransactions
```

And now you also can check your wallet scyptocurrency balance. You minted 3 blocks , so got 30 coins! By default, 10 coins per block are awarded to a block maker.

```
./node getbalances
```

And now try to send your coins to some other wallet

```
# create one more wallet
./node createwallet
./node send -from YOUR_WALLET_ADDRESS -to YOUR_NEW_WALLET_ADDRESS -amount 5
./node getbalances
# your transaction is not yet approved, no enough transactions to make a block. Make other 2 transactions
./node send -from YOUR_WALLET_ADDRESS -to YOUR_NEW_WALLET_ADDRESS -amount 1
./node send -from YOUR_WALLET_ADDRESS -to YOUR_NEW_WALLET_ADDRESS -amount 1
# after some time new block will be created and transactions confirmed
./node getbalances
```

## Multiple nodes (cluster)

Real value of this tool is decentralization . When many different people run nodes.

You can test multiple nodes on a single machine. For this you need to make separate mysql DB and keep config in diffferent directory. You can execute same ./node command but add new argument -configdir=PATH, for example, -configdir=conf2 . And new DB is BC2

Presume your first node is already running and listens on localhost:8765

Prepare other node. Create a wallet first. And import blockchain (connect to existent blockchain network).

```
./node createwallet -configdir=conf2
./node importblockchain -configdir=conf2 -nodeaddress localhost:8765 -mysqluser blockchain -mysqlpass blockchain -mysqldb BC2
# update settings and run the node
# YOUR_WALLET_ADDRESS is an address created with the createwallet command
./node updateconfig -configdir=conf2 -minter YOUR_WALLET_ADDRESS -port 9765 -host localhost -dbproxyaddr :9766 -proxykey YOUR_WALLET_ADDRESS
./node startnode -configdir=conf2
# check blockchein on this node
./node printchain -configdir=conf2
```

Now do update on the second node and check the state on the first node. Check both nodes DB first

```
# first node
mysql -h 127.0.0.1 -P 8766 -u blockchain -pblockchain BC
> SELECT * FROM test;
# second node
mysql -h 127.0.0.1 -P 9766 -u blockchain -pblockchain BC2
> SELECT * FROM test;
> INSERT INTO test SET b='row3';
# first node
mysql -h 127.0.0.1 -P 8766 -u blockchain -pblockchain BC
> SELECT * FROM test;
> UPDATE test SET b='row3_updated' WHERE a=1;
# second node
mysql -h 127.0.0.1 -P 9766 -u blockchain -pblockchain BC2
> SELECT * FROM test;
```

You could do some more tests. Try to add more nodes, try to stop some nodes, do updates, see how SQL conflicts are resolved.

You can disconnect nodes from network using commands shownodes,addnode,removenode .



