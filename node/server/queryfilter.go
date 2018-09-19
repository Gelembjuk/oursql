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

	"github.com/gelembjuk/oursql/lib/dbproxy"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/nodemanager"
)

type queryFilter struct {
	DBProxy             dbproxy.DBProxyInterface
	Node                *nodemanager.Node
	Logger              *utils.LoggerMan
	sessionTransactions map[string][]byte
	// Use this to notify a main server process about new transaction was added to a pool
	newTransactionChan chan []byte
}

func InitQueryFilter(proxyAddr, dbAddr string, node *nodemanager.Node, logger *utils.LoggerMan, newTXChan chan []byte) (q *queryFilter, err error) {
	q = &queryFilter{}

	q.Logger = logger
	q.Node = node
	q.sessionTransactions = make(map[string][]byte)
	q.newTransactionChan = newTXChan

	q.Logger.Trace.Printf("DB Proxy Start on %s  %s", proxyAddr, dbAddr)

	q.DBProxy, err = dbproxy.NewMySQLProxy(proxyAddr, dbAddr)

	if err != nil {
		return
	}

	q.DBProxy.SetLoggers(q.Logger.Trace, q.Logger.Error)

	q.DBProxy.SetFilter(q)

	err = q.DBProxy.Init()

	if err != nil {
		return
	}

	err = q.DBProxy.Run()

	q.Logger.Trace.Println("DB proxy started")

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
		if result.ErrorCode > 0 {
			return dbproxy.NewCustomErrorResponse(result.Error.Error(), result.ErrorCode), nil
		}
		return nil, result.Error
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

		q.sessionTransactions[sessionID] = result.TX.GetID()

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

		if txID, ok := q.sessionTransactions[sessionID]; ok {
			err := q.Node.GetTransactionsManager().CancelTransaction(txID, false)

			if err != nil {
				q.Logger.Trace.Printf("DB Proxy Can Not Cancel TX: %s.", err.Error())
			}
		}

	} else if txID, ok := q.sessionTransactions[sessionID]; ok {
		// Notify server thread about new TX completed fine

		// non-blocking sending to a channel
		select {
		case q.newTransactionChan <- txID: // notify server thread about new transaction added to the pool
		default:
		}
		q.Logger.Trace.Printf("Sent to channel TX %x\n", txID)

	}
}
func (q *queryFilter) Stop() error {
	q.Logger.Trace.Println("Stop DB proxy")

	q.DBProxy.Stop()

	q.Logger.Trace.Println("DB proxy stopped")

	return nil
}
