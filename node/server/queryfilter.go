package server

/*
Error codes
Errors returned by the proxy must have MySQL codes.
2 - Query requires public key
3 - Query requires data to sign
4 - Error preparing of query parsing

*/
import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gelembjuk/oursql/lib"
	"github.com/gelembjuk/oursql/lib/dbproxy"
	"github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/lib/remoteclient"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/dbquery"
	"github.com/gelembjuk/oursql/node/nodemanager"
	"github.com/gelembjuk/oursql/node/structures"
)

const (
	internalCommandTableNodes        = "knownnodes"
	internalCommandTableTransactions = "transactions"
	internalCommandTableBlocks       = "blocks"
	internalCommandTableWallets      = "keys"
	internalCommandTableServer       = "nodestate"
)

type queryFilter struct {
	DBProxy             dbproxy.DBProxyInterface
	Node                *nodemanager.Node
	Logger              *utils.LoggerMan
	sessionTransactions map[string]*structures.Transaction
	// Use this to notify a main server process about new transaction was added to a pool
	newTransactionChan chan []byte
	blockmakerObj      *blocksMaker
	nodeAuthStr        string
}

func InitQueryFilter(proxyAddr, dbAddr string, node *nodemanager.Node,
	logger *utils.LoggerMan, bmo *blocksMaker, nodeAuthStr string) (q *queryFilter, err error) {
	q = &queryFilter{}

	q.Logger = logger
	q.Node = node
	q.sessionTransactions = make(map[string]*structures.Transaction)
	q.blockmakerObj = bmo

	q.nodeAuthStr = nodeAuthStr

	q.Logger.Trace.Printf("DB Proxy Start on %s  %s", proxyAddr, dbAddr)

	q.DBProxy, err = dbproxy.NewMySQLProxy(proxyAddr, dbAddr)

	if err != nil {
		q.Logger.Error.Printf("Error DB proxy start %s", err.Error())
		return
	}

	q.DBProxy.SetLoggers(q.Logger.Trace, q.Logger.Error)

	q.DBProxy.SetFilter(q)

	err = q.DBProxy.Init()

	if err != nil {
		q.Logger.Error.Printf("Error DB proxy init %s", err.Error())
		return
	}

	err = q.DBProxy.Run()

	if err != nil {
		q.Logger.Error.Printf("Error DB proxy run %s", err.Error())
	} else {
		q.Logger.Trace.Println("DB proxy started")
	}

	return
}
func (q *queryFilter) RequestCallback(query string, sessionID string) (dbproxy.CustomRequestActionInterface, error) {
	qm, err := q.Node.GetSQLQueryManager()

	if err != nil {
		return nil, err
	}
	result := qm.NewQueryFromProxy(query)

	q.Logger.Trace.Printf("Proxy Query process status %d", result.Status)

	if result.Error != nil {
		q.Logger.Trace.Printf("Proxy Query error %s code %d", result.Error.Error(), result.ErrorCode)
		if result.ErrorCode > 0 {
			return dbproxy.NewCustomErrorResponse(result.Error.Error(), result.ErrorCode), nil
		}
		return nil, result.Error
	}

	if result.Status == 4 {
		// It is internal command. We need to do somethign and return data to user.

		response, err := q.internalCommand(result.ParsedInfo)

		if err != nil {
			return nil, err
		}

		if response == nil {
			//q.Logger.TraceExt.Printf("Return OK custom response")
			// return OK
			return dbproxy.NewCustomOKResponse(1, 1), nil
		}
		return dbproxy.NewCustomDataKeyValueResponse(response), nil
	}

	if result.Status == 2 {
		// return prepared signature data
		response := []dbproxy.CustomResponseKeyValue{}

		response = append(response, dbproxy.CustomResponseKeyValue{"Transaction", hex.EncodeToString(result.TXData)})

		response = append(response, dbproxy.CustomResponseKeyValue{"StringToSign", hex.EncodeToString(result.StringToSign)})

		q.Logger.Trace.Println("Return transaction prepare info")

		return dbproxy.NewCustomDataKeyValueResponse(response), nil
	}

	if result.TX != nil {
		q.Logger.Trace.Printf("Query: %s, sessID: %s, TX created %x\n", query, sessionID, result.TX.GetID())

		q.sessionTransactions[sessionID] = result.TX

	} else {
		q.Logger.Trace.Printf("Query: %s, sessID: %s, no TX needed\n", query, sessionID)
	}

	if result.Status == 3 {
		// it means query was not executed and must be passed to a server
		return nil, nil
	}
	// else
	// query was already executed and we only have to return OK response
	// empty list of rows means to return OK response
	return dbproxy.NewCustomQueryRequest(result.ReplaceQuery), nil
}
func (q *queryFilter) ResponseCallback(sessionID string, err error) {

	if err != nil {
		q.Logger.Trace.Printf("DB Proxy Response Error: %s. Canceling TX from a pool", err.Error())

		if _, ok := q.sessionTransactions[sessionID]; ok {
			delete(q.sessionTransactions, sessionID)
		}

	} else if tx, ok := q.sessionTransactions[sessionID]; ok {
		// Add the TX to the pool
		err := q.Node.ReceivedNewTransaction(tx, lib.TXFlagsVerifyAllowMissedForDelete)

		if err != nil {
			// Rollback?
			// TODO
			q.Logger.Trace.Printf("Error adding TX to pool from proxy %x %s", tx.GetID(), err.Error())
		}

		// Notify server thread about new TX completed fine

		q.blockmakerObj.NewTransaction(tx.GetID())

		delete(q.sessionTransactions, sessionID)
	}
}
func (q *queryFilter) Stop() error {
	q.Logger.Trace.Println("Stop DB proxy")

	if q.DBProxy != nil {
		q.DBProxy.Stop()

		q.Logger.Trace.Println("DB proxy stopped")
	}

	return nil
}

// Process internal command made with pseudo SQL
func (q *queryFilter) internalCommand(parsedInfo dbquery.QueryParsed) ([]dbproxy.CustomResponseKeyValue, error) {
	// check AUTH string

	if parsedInfo.InternalAuth == "" {
		return nil, errors.New("Internal request has no auth")
	}

	if parsedInfo.InternalAuth != q.nodeAuthStr {
		return nil, errors.New("Internal request auth str is not valid")
	}

	switch parsedInfo.Structure.GetTable() {
	// NODES command. shownodes, addnode, removenode
	case internalCommandTableNodes:
		return q.internalCommandNodes(parsedInfo)

	// Blocks command. printchain
	case internalCommandTableBlocks:
		return q.internalCommandBlocks(parsedInfo)

	// Transactions command. unapprovedtransactions, addrhistory, send
	case internalCommandTableTransactions:
		return q.internalCommandTransactions(parsedInfo)

	// Wallets command. createwallet, listaddresses, getbalance, getbalances
	case internalCommandTableWallets:
		return q.internalCommandWallets(parsedInfo)

	// Wallets command. nodestate
	case internalCommandTableServer:
		return q.internalCommandServer(parsedInfo)

	default:
		return nil, errors.New("Unknown internal command")
	}
	return nil, nil
}

// Process nodes command
func (q *queryFilter) internalCommandNodes(parsedInfo dbquery.QueryParsed) ([]dbproxy.CustomResponseKeyValue, error) {

	if parsedInfo.Structure.GetKind() == lib.QueryKindSelect {
		// return list of nodes
		nodes := q.Node.NodeNet.GetNodes()

		response := []dbproxy.CustomResponseKeyValue{}
		q.Logger.Trace.Printf("%d known nodes", len(nodes))

		for _, n := range nodes {
			q.Logger.Trace.Printf("Known node: %s", n.NodeAddrToString())
			response = append(response, dbproxy.CustomResponseKeyValue{n.NodeAddrToString(), ""})
		}
		return response, nil
	}

	if parsedInfo.Structure.GetKind() == lib.QueryKindInsert {

		cl := parsedInfo.Structure.GetUpdateColumns()

		addr := ""

		if a, ok := cl["address"]; ok {
			addr = a
		} else {
			return nil, errors.New("Address field is not found")
		}
		// add new node
		newaddr := net.NodeAddr{}
		err := newaddr.LoadFromString(addr)

		if err != nil {
			return nil, err
		}
		q.Logger.Trace.Printf("Add to known nodes %s", newaddr.NodeAddrToString())
		q.Node.AddNodeToKnown(newaddr, false)
	}

	if parsedInfo.Structure.GetKind() == lib.QueryKindDelete {
		// delete node
		column, val := parsedInfo.Structure.GetOneColumnCondition()

		if column == "" {
			return nil, errors.New("Address not found in request")
		}

		if column != "address" {
			return nil, errors.New("Wrong condition column")
		}

		if val == "" {
			return nil, errors.New("Address value missed")
		}

		remaddr := net.NodeAddr{}
		err := remaddr.LoadFromString(val)

		if err != nil {
			return nil, err
		}
		q.Logger.Trace.Printf("Remove node from known %s", remaddr.NodeAddrToString())
		q.Node.NodeNet.RemoveNodeFromKnown(remaddr)
	}

	return nil, nil
}

// Process blocks command
func (q *queryFilter) internalCommandBlocks(parsedInfo dbquery.QueryParsed) ([]dbproxy.CustomResponseKeyValue, error) {
	if parsedInfo.Structure.GetKind() != lib.QueryKindSelect {
		return nil, errors.New("Unsupported query")
	}
	bci, err := q.Node.GetBlockChainIterator()

	if err != nil {
		return nil, err
	}
	response := []dbproxy.CustomResponseKeyValue{}

	for {
		blockfull, err := bci.Next()

		if err != nil {
			return nil, err
		}

		if blockfull == nil {
			fmt.Printf("Somethign went wrong. Next block can not be loaded\n")
			break
		}
		block := blockfull.GetSimpler()

		response = append(response, dbproxy.CustomResponseKeyValue{fmt.Sprintf("%x", blockfull.Hash), block.String()})

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return response, nil
}

// Process transactions command
func (q *queryFilter) internalCommandTransactions(parsedInfo dbquery.QueryParsed) ([]dbproxy.CustomResponseKeyValue, error) {
	if parsedInfo.Structure.GetKind() == lib.QueryKindInsert {
		// SEND currency

		cl := parsedInfo.Structure.GetUpdateColumns()

		from := ""
		to := ""
		amount := float64(0)

		if a, ok := cl["from"]; ok {
			from = a
		} else {
			return nil, errors.New("From address field is not found")
		}

		if a, ok := cl["to"]; ok {
			to = a
		} else {
			return nil, errors.New("TO address field is not found")
		}

		if a, ok := cl["amount"]; ok {
			var err error
			amount, err = strconv.ParseFloat(a, 64)

			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("TO address field is not found")
		}

		wallets := remoteclient.NewWallets(q.Node.ConfigDir)
		err := wallets.LoadFromFile()

		if err != nil {
			return nil, err
		}

		walletobj, err := wallets.GetWallet(from)

		if err != nil {
			return nil, err
		}

		_, err = q.Node.Send(walletobj.GetPublicKey(), walletobj.GetPrivateKey(),
			to, amount)

		if err != nil {
			return nil, err
		}

		return nil, err
	}
	if parsedInfo.Structure.GetKind() != lib.QueryKindSelect {
		return nil, errors.New("Unknown query")
	}

	column, val := parsedInfo.Structure.GetOneColumnCondition()

	if column == "" {
		return nil, errors.New("Unknown query")
	}

	if column == "state" && val == "PENDING" {
		// return list of pending transactions
		response := []dbproxy.CustomResponseKeyValue{}

		q.Node.GetTransactionsManager().ForEachUnapprovedTransaction(
			func(txhash, txstr string) error {

				response = append(response, dbproxy.CustomResponseKeyValue{fmt.Sprintf("%x", txhash), txstr})
				return nil
			})
		return response, nil
	}
	if column == "currencyaddr" {
		// get history of this addr
		result, err := q.Node.NodeBC.GetAddressHistory(val)

		if err != nil {
			return nil, err
		}
		response := []dbproxy.CustomResponseKeyValue{}

		for _, rec := range result {
			if rec.IOType {
				response = append(response, dbproxy.CustomResponseKeyValue{fmt.Sprintf("%f", rec.Value), fmt.Sprintf("%f\t In from\t%s\n", rec.Value, rec.Address)})

			} else {
				response = append(response, dbproxy.CustomResponseKeyValue{fmt.Sprintf("%f", rec.Value), fmt.Sprintf("%f\t Out To  \t%s\n", rec.Value, rec.Address)})
			}

		}
		return response, nil
	}

	return nil, errors.New("Unknown query")
}

// Process wallets command
func (q *queryFilter) internalCommandWallets(parsedInfo dbquery.QueryParsed) ([]dbproxy.CustomResponseKeyValue, error) {
	wallets := remoteclient.NewWallets(q.Node.ConfigDir)
	err := wallets.LoadFromFile()

	if err != nil {
		return nil, err
	}

	if parsedInfo.Structure.GetKind() == lib.QueryKindInsert {
		// create new wallet and return OK

		_, err := wallets.CreateWallet()

		return nil, err
	}
	if parsedInfo.Structure.GetKind() != lib.QueryKindSelect {
		return nil, errors.New("Unknown query")
	}

	response := []dbproxy.CustomResponseKeyValue{}

	if !strings.Contains(parsedInfo.SQL, "balance") {
		// return only list of addresses

		for _, addr := range wallets.GetAddresses() {
			mod := ""

			if q.Node.MinterAddress == addr {
				mod = "minter"
			}
			if len(q.Node.ProxyPubKey) > 0 {
				proxyaddr, _ := utils.PubKeyToAddres(q.Node.ProxyPubKey)

				if proxyaddr == addr {
					if mod != "" {
						mod = mod + ","
					}
					mod = mod + "proxy"
				}
			}

			response = append(response, dbproxy.CustomResponseKeyValue{addr, mod})
		}
	} else {
		// if has a condition, it is request for particular balance
		column, addr := parsedInfo.Structure.GetOneColumnCondition()

		if column == "address" {
			// particular balance
			balance, err := q.Node.GetTransactionsManager().GetAddressBalance(addr)

			if err != nil {
				return nil, err
			}
			response = append(response, dbproxy.CustomResponseKeyValue{addr, strconv.FormatFloat(balance.Total, 'f', -1, 64)})
		} else {
			// all balances
			for _, addr := range wallets.GetAddresses() {
				balance, err := q.Node.GetTransactionsManager().GetAddressBalance(addr)

				if err != nil {
					return nil, err
				}
				response = append(response, dbproxy.CustomResponseKeyValue{addr, strconv.FormatFloat(balance.Total, 'f', -1, 64)})
			}
		}
	}

	return response, nil
}

// Process node server state command
func (q *queryFilter) internalCommandServer(parsedInfo dbquery.QueryParsed) ([]dbproxy.CustomResponseKeyValue, error) {
	// there can be only 1 case - SELECT * FROM nodestate
	info, err := q.Node.GetNodeState()

	if err != nil {
		return nil, err
	}

	response := []dbproxy.CustomResponseKeyValue{}

	response = append(response, dbproxy.CustomResponseKeyValue{"NumberOfBlocks", strconv.Itoa(info.BlocksNumber)})

	response = append(response, dbproxy.CustomResponseKeyValue{"TransactionsInPool", strconv.Itoa(info.TransactionsCached)})

	return response, nil
}
