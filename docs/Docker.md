# OurSQL docker container

Usage examples.

Docker image is named oursql/oursql-server

## Pull the image

```
docker pull oursql/oursql-server
```

Or just try to run, it will pull automatically.

```
docker run --name oursql -p 8766:8766 -d -it oursql/oursql-server
```

The OurSQL image exposes 2 ports:

* 8765 - node server port to communicate with other nodes and client applications

## Single node

```
docker run --name oursql -p 8766:8766 -d -it oursql/oursql-server
```

Now you can connect with mysql client and execute SQL updates.

```
mysql -h 127.0.0.1 -P 8766 -u blockchain -pblockchain BC
> CREATE TABLE test (a int unsigned primary key auto_increment, b varchar(100));
> INSERT INTO test SET b='row1';
```

To check that blocks are created execute the command
```
# to see list of blocks
docker exec -it oursql ./node printchain

# to see transactions pool (transactions that are not yet in blocks)
docker exec -it oursql ./node unapprovedtransactions

# node state info
docker exec -it oursql ./node nodestate
```

Additionally, you can play with cryptocurrency transactions:

```
# list your wallets with balance of money on it
docker exec -it oursql ./node getbalances

# create new wallet
docker exec -it oursql ./node createwallet

# check balances again and send money from your first wallet to your new wallet
# (in your case addresses will be different)
docker exec -it oursql ./node send -from 1Me...6Hx -to 1NT...rko -amount 5

# check balances. Execute some more tranactions (SQL with mysql or send money) to get new block
docker exec -it oursql ./node getbalances
OurSQL - 0.1.1 beta

Balance for all addresses:

1NT...rko: 5.00000000 (Approved - 5.00000000, Pending - 0.00000000)
1Me...6Hx: 35.00000000 (Approved - 35.00000000, Pending - 0.00000000)

```

## Run multiple nodes

More interesting test is when you run multiple nodes. This shows how the systtem works.

### Start node and create new blockchain

```
docker run --name oursql1 -p 9001:8765 -p 9002:8766  -d -it oursql/oursql-server interactiveautocreate -port 9001
```

This command maps 2 ports. 9001 - is the port on where a node listens connections from other nodes.
9002 is a DB proxy port. "interactiveautocreate" is a command to start new blockchain and run a node. "-port 9001" says to a node "your external port is 9001".

Do some SQL updates

```
mysql -h 127.0.0.1 -P 9002 -u blockchain -pblockchain BC
> CREATE TABLE test (a int unsigned primary key auto_increment, b varchar(100));
> INSERT INTO test SET b='row1';
``` 

### Start second node and import blockchain from the first node 

```
docker run --name oursql2 -p 9003:8765 -p 9004:8766  -d -it oursql/oursql-server importfromandstart -port 9003 -nodeaddress host.local.address:9001 
```

Ports of the second node are: 9003 - node server port, 9004 - proxy server port.

"-nodeaddress host.local.address:9001" means - connect to the node on the host host.local.address and port 9001 and import blockchain from here.

"host.local.address" is a hostname of a host machine from inside a docker container.

Now your 2 nodes are connected in a cluster.

Connect to your second node and you would see a table created on a first node.

```
mysql -h 127.0.0.1 -P 9004 -u blockchain -pblockchain BC
> SELECT * FROM test
```

Execute SQL commands on two DB proxy servers and see how data are replicated!

### Start a blockchain with custom consensus config

Use the option -consensusfile PATH_TO_CONFIG to get a cutom consensus rules, for exmaple, more complex mining or different amount of coins for a minter per a block. See more about [consensus config](Consensus.md)

```
docker run --name oursql1 -p 9001:8765 -p 9002:8766 -v $(pwd)/path/to/consconf.json:/cons.json -d -it oursql/oursql-server interactiveautocreate -port 9001 -consensusfile /cons.json
```

$(pwd)/path/to/consconf.json is the path to a config file on a host file system. Must be an absolute path.

