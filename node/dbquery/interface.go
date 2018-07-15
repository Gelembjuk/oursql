package dbquery

import (
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
)

const (
	QueryKindSelect = "select"
	QueryKindUpdate = "update"
	QueryKindInsert = "insert"
	QueryKindDelete = "delete"
	QueryKindCreate = "create"
	QueryKindOther  = "other"
)

type QueryProcessorInterface interface {
	ParseQuery(sqlquery string) (QueryParsed, error)
	CheckQuerySyntax(sql string) error
	ExecteQuery(sql string) error
	FormatSpecialErrorMessage(errorKind uint, txdata []byte, datatosign []byte) (string, error)
	MakeSQLUpdateStructure(sql string) (structures.SQLUpdate, error)
}

func NewQueryProcessor(DB database.DBManager, Logger *utils.LoggerMan) QueryProcessorInterface {
	return &queryProcessor{DB, Logger}
}
