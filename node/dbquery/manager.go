package dbquery

import (
	"crypto/ecdsa"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/structures"

	"github.com/gelembjuk/oursql/node/database"
)

type mySQLQueryManager struct {
	DB     database.DBManager
	Logger *utils.LoggerMan
}

func NewManager(DB database.DBManager, Logger *utils.LoggerMan) DbQueryInterface {
	return &mySQLQueryManager{DB, Logger}
}

func (dbq *mySQLQueryManager) StartSQLQueryTransaction(PubKey []byte, privKey ecdsa.PrivateKey, sqlcommand string) (*structures.Transaction, error) {
	return nil, nil
}
