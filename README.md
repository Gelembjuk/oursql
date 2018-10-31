# OurSQL

![OurSQL](docs/oursql_logo_300.png?raw=true "OurSQL Logo")

Project Web site [http://oursql.org](http://oursql.org)

## Blockchain replication layer for MySQL

OurSQL allows to create a blockchain database quickly with connecting MySQL servers into a cluster.

## How  It Works

![How OurSQL Works](docs/oursql_how_it_works.png?raw=true "OurSQL")

## Consensus

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

Find more usage [examples](docs/README.md) and try to run multiple nodes to see how replication works.

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

Find usage [examples](docs/README.md) to do some tests.

### Install

We don't have intallation packages yet. We plan to make it soon. 

## Author

Roman Gelembjuk , roman@gelembjuk.com 

http://gelembjuk.com/

### More info about OurSQL and contacts

[http://oursql.org](http://oursql.org)  

Email: oursql.project@gmail.com

[Twitter: OursqlO](https://twitter.com/OursqlO)

## License

GNU3 