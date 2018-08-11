package dbquery

import (
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
)

type QueryProcessorInterface interface {
	ParseQuery(sqlquery string) (QueryParsed, error)
	ExecuteQuery(sql string) (*structures.SQLUpdate, error)
	ExecuteParsedQuery(qp QueryParsed) (*structures.SQLUpdate, error)
	ExecuteQueryFromTX(sql structures.SQLUpdate) error
	ExecuteRollbackQueryFromTX(sql structures.SQLUpdate) error
	FormatSpecialErrorMessage(errorKind uint, txdata []byte, datatosign []byte) (string, error)
	MakeSQLUpdateStructure(parsed QueryParsed) (structures.SQLUpdate, error)
}

func NewQueryProcessor(DB database.DBManager, Logger *utils.LoggerMan) QueryProcessorInterface {
	return &queryProcessor{DB, Logger}
}
