# OurSQL Consensus Configuration

Current version supports only Proof of Work consensus type. Every blockchain has a consensus config file which containes rules. Options of PoW: block hash options, coins to add for minter, numbers of transactions per block etc.

This file is distributed as part of a package instalation package.

Also, in this file it is possible to describe which SQL operation are allowed in this decetralized DB: insert, update, delete , table create. Additionally, it is possible to set rules per table.

Finally, it is possible to set a cost of SL query per table and type. For example, 0.5 (of a coin) per insert in a table "members". This allows to control updates.

## Future support of consensus
Soon we add extended support of a consensus management.

We are going to add support of a consensus plug-in. It will be a module for OurSQL (.so or .dll) to control updates.

The consensus module will be able to do some work for every proposed update. Each SQL transaction goes together with a wallet address. The module will be able to check if an address can do this SQL command now. It can do extra requests to other DB table to do some checks, etc.

For example, all users of an app can vote for some user to be a moderator. If a user was elected he is able to update some table. All other are not able. Consensus module can controll such things.

Consensus module will filter all SQL commands received from an app via the proxy and also received from other nodes.

## Conseusus config file for Proof Of Work

The config is a JSON file.

```
{
    "Application": {
        "Name":"MyBlockchainApp",
        "WebSite":"http://oursql.org",
        "Team":"Application Support Team"
    },
    "Kind":"proofofwork",
    "CoinsForBlockMade":10.0,
    "Settings":{
        "Complexity":16,
        "ComplexityStep2":24,
        "MaxMinNumberTransactionInBlock":1000,
        "MaxNumberTransactionInBlock":10000
    },
    "AllowTableCreate":true,
    "AllowTableDrop":true,
    "AllowRowDelete":true,
    "TransactionCost":{
        "Default":0.0,
        "RowDelete":0.0,
        "RowUpdate":0.0,
        "RowInsert":0.0,
        "TableCreate":0.0
    },
    "UnmanagedTables":[],
    "TableRules":[
        {
            "Table":"members",
            "AllowRowDelete":false,
            "AllowRowUpdate":true,
            "AllowRowInsert":true,
            "AllowTableCreate":true,
            "TransactionCost":{
                "Default":0.05,
                "TableCreate":1
            }
        }
    ],
    "InitNodesAddreses":["startnodehost:8765"]
}
```

On a blockchain creation you need to point OurSQL to the consensus config file using the argument -consensusfile FILEPATH

```
./node initblockchain -minter YOUR_WALLET_ADDRESS  -mysqluser blockchain -mysqlpass blockchain -mysqldb BC -consensusfile PATH_TO_CONSENSUS_CONFIG
```
### Proof of Work settings

*Complexity* - integer number, defines how much workis it required to make a hash. This number is a number of 0 bits on the beginning of the hash. 16 means there will be 2 "0" at the beginning (2 zero bytes). 24 will be more complex as require 3 zero bytes.

*ComplexityStep2* - optional argument, if is set , its value replaces Complexity after acheaving number of blocks defined in MaxMinNumberTransactionInBlock

*MaxMinNumberTransactionInBlock* - when a blockchain is created, minimum of transactions per block is same as a block height (block number). For 3-th block it is required 3 transactions minimum, for 4-th 4 and so on. When height of a block acheaves MaxMinNumberTransactionInBlock value, minimum number of transactions just should not be les this value (for example, minimum 1000 tranactions per block after 1000-th block).

*MaxNumberTransactionInBlock* - maximum number of transactions per block

### SQL updates settings

There are common settings for all tables and optional custom settings for each table.

* AllowRowDelete - allow to delete rows or no
* AllowRowUpdate - allow to update table rows or no
* AllowRowInsert - allow to insert new rows in a table
* AllowTableCreate - allow to create tables
* TransactionCost - SQL operation cost. Has default value or custom per operation. Value is in internal cryptocrrency

#### Skipping some tables

There can be tables in a DB which are not required to sync between nodes. TO keep some local data. Such tables can be just listed in an array.

```
"UnmanagedTables":["tmp","seenposts"],
```

#### Address of the initial node

```
"InitNodesAddreses":["startnodehost.org:8765"]
```

This is the array of TCP addresses in the format "host:port". It is the list of nodes to import blockchain for fresh intalled nodes.