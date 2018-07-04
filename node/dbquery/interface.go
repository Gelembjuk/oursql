package dbquery

import (
	"crypto/ecdsa"

	"github.com/gelembjuk/oursql/node/structures"
)

type DbQueryInterface interface {
	StartSQLQueryTransaction(PubKey []byte, privKey ecdsa.PrivateKey, sqlcommand string) (*structures.Transaction, error)
}
