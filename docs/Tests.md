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

### Single node

### Multiple nodes (cluster)

