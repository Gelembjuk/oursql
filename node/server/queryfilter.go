package server

import (
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
func (q *queryFilter) RequestCallback(query string, sessionID string) error {
	qm, err := q.Node.GetSQLQueryManager()

	if err != nil {
		return err
	}
	tx, err := qm.NewQueryFromProxy(query)

	if err != nil {
		return err
	}
	if tx != nil {
		q.Logger.Trace.Printf("Query: %s, sessID: %s, TX created %x\n", query, sessionID, tx.GetID())
	} else {
		q.Logger.Trace.Printf("Query: %s, sessID: %s, no TX needed\n", query, sessionID)
	}

	return nil
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
