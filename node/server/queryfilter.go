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
	DBProxy        dbproxy.DBProxyInterface
	Node           *nodemanager.Node
	Logger         *utils.LoggerMan
	sessionQueries map[string]string
}

func InitQueryFilter(proxyAddr, dbAddr string, node *nodemanager.Node, logger *utils.LoggerMan) (q *queryFilter, err error) {
	q = &queryFilter{}

	q.Logger = logger
	q.Node = node
	q.sessionQueries = make(map[string]string)

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
func (q *queryFilter) RequestCallback(query string, sessionID string) ([]dbproxy.CustomResponseKeyValue, error) {
	qm, err := q.Node.GetSQLQueryManager()

	if err != nil {
		return nil, err
	}
	result := qm.NewQueryFromProxy(query)

	q.Logger.Trace.Printf("Proxy Query process status %d", result.Status)

	if result.Error != nil {
		if result.ErrorCode > 0 {
			return nil, dbproxy.NewMySQLError(result.Error.Error(), result.ErrorCode)
		}
		return nil, result.Error
	}

	if result.Status == 2 {
		// return prepared signature data
		response := []dbproxy.CustomResponseKeyValue{}

		response = append(response, dbproxy.CustomResponseKeyValue{"Transaction", hex.EncodeToString(result.TXData)})

		response = append(response, dbproxy.CustomResponseKeyValue{"StringToSign", hex.EncodeToString(result.StringToSign)})

		q.Logger.Trace.Println("Return transaction prepare info")

		return response, nil
	}

	if result.TX != nil {
		q.Logger.Trace.Printf("Query: %s, sessID: %s, TX created %x\n", query, sessionID, result.TX.GetID())
	} else {
		q.Logger.Trace.Printf("Query: %s, sessID: %s, no TX needed\n", query, sessionID)
	}

	return nil, nil
}
func (q *queryFilter) ResponseCallback(sessionID string, err error) {

	if err != nil {
		q.Logger.Trace.Printf("DB Proxy Response Error: %s\n", err.Error())
	}
}
func (q *queryFilter) Stop() error {
	q.Logger.Trace.Println("Stop DB proxy")

	q.DBProxy.Stop()

	q.Logger.Trace.Println("DB proxy stopped")

	return nil
}
