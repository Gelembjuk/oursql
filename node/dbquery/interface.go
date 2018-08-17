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

type SQLUpdateInterface interface {
	CheckUpdateCanFollow(sqlUpdPrev *structures.SQLUpdate) error
	CheckAllowsMultipleSubtransactions(sqlUpdPrev *structures.SQLUpdate) (bool, error)
	GetAlternativeRefID() ([]byte, error)
	RequiresBaseTransation() bool
}

func NewQueryProcessor(DB database.DBManager, Logger *utils.LoggerMan) QueryProcessorInterface {
	return &queryProcessor{DB, Logger}
}

func NewSQLUpdateManager(SQLUpdate structures.SQLUpdate) (SQLUpdateInterface, error) {
	o := sqlUpdateManager{SQLUpdate, nil}

	err := o.Init()

	if err != nil {
		return nil, err
	}

	return &o, nil
}
