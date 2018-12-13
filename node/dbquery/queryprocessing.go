package dbquery

import (
	"errors"
	"fmt"

	"github.com/gelembjuk/oursql/lib"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/dbquery/sqlparser"
	"github.com/gelembjuk/oursql/node/structures"
)

var primaryKeysCache map[string]string

type queryProcessor struct {
	DB     database.DBManager
	Logger *utils.LoggerMan
}

// checks if this query is syntax correct , return altered query if needed
func (qp queryProcessor) ParseQuery(sqlquery string) (r QueryParsed, err error) {
	r.Structure = sqlparser.NewSqlParser()

	err = r.Structure.Parse(sqlquery)

	if err != nil {
		return
	}

	// check syntax
	err = qp.checkQuerySyntax(r.Structure)

	if err != nil {
		return
	}

	// this will extract key column, its value, check if it is present
	err = qp.patchRowInfo(&r)

	if err != nil {
		return
	}

	r.PubKey, r.Signature, r.TransactionBytes, err = r.parseInfoFromComments()

	if err != nil {
		return
	}

	r.SQL = r.Structure.GetCanonicalQuery()

	return r, nil
}

// checks if this query is syntax correct
func (qp queryProcessor) checkQuerySyntax(sqlparsed sqlparser.SQLQueryParserInterface) error {
	if sqlparsed.GetKind() == lib.QueryKindInsert ||
		sqlparsed.GetKind() == lib.QueryKindDelete ||
		sqlparsed.GetKind() == lib.QueryKindUpdate {

		_, err := qp.DB.QM().ExecuteSQLExplain(sqlparsed.GetCanonicalQuery())

		if err != nil {
			return errors.New(fmt.Sprintf("Syntax check error: %s", err.Error()))
		}
	}

	return nil
}

// return info for a row that will be affected by a query. If that is update or delete
// return a row
// if it is insert, try to get next autoincrement
func (qp queryProcessor) patchRowInfo(parsed *QueryParsed) (err error) {
	if parsed.Structure.GetKind() != lib.QueryKindUpdate &&
		parsed.Structure.GetKind() != lib.QueryKindDelete &&
		parsed.Structure.GetKind() != lib.QueryKindInsert {
		return
	}
	var keyCol string

	if primaryKeysCache != nil {
		if k, ok := primaryKeysCache[parsed.Structure.GetTable()]; ok {
			keyCol = k
		}
	}
	if keyCol == "" {
		keyCol, err = qp.DB.QM().ExecuteSQLPrimaryKey(parsed.Structure.GetTable())

		if err != nil {
			return
		}
		if primaryKeysCache == nil {
			primaryKeysCache = make(map[string]string, 0)
		}
		primaryKeysCache[parsed.Structure.GetTable()] = keyCol
	}

	parsed.KeyCol = keyCol

	if parsed.Structure.GetKind() == lib.QueryKindUpdate ||
		parsed.Structure.GetKind() == lib.QueryKindDelete {

		cKey, cVal := parsed.Structure.GetOneColumnCondition()

		if cKey != keyCol {
			err = errors.New("Query condition has no a primary key")
			return
		}

		sqlquery := "SELECT * FROM " + parsed.Structure.GetTable() + " WHERE " + keyCol + "='" + database.Quote(cVal) + "'"

		var currentRow map[string]string

		currentRow, err = qp.DB.QM().ExecuteSQLSelectRow(sqlquery)

		if err != nil {
			return
		}

		parsed.RowBeforeQuery = currentRow
		parsed.KeyVal = cVal

	} else if parsed.Structure.GetKind() == lib.QueryKindInsert {
		// there can be different primary key and it can be in list of insert columns

		cols := parsed.Structure.GetUpdateColumns()

		if val, ok := cols[keyCol]; ok {
			parsed.KeyVal = val

			return
		}
		// try to predict key value
		// try to get next auto_increment
		var nextID string
		nextID, err = qp.DB.QM().ExecuteSQLNextKeyValue(parsed.Structure.GetTable())

		if err != nil {
			return
		}

		if nextID == "" {
			err = errors.New("Can not build reference ID for inserted row. Table has no auto_increment key")
			return
		}

		err = parsed.Structure.ExtendInsert(keyCol, nextID, "string")

		if err != nil {
			return
		}

		parsed.KeyVal = nextID
		parsed.SQL = parsed.Structure.GetCanonicalQuery()

	}
	// do extra verification.
	// we don't allow to change a key column value with UPDATE query. It can break the system

	if parsed.Structure.GetKind() == lib.QueryKindUpdate {
		if val, ok := parsed.Structure.GetUpdateColumns()[keyCol]; ok {
			if val != keyCol {
				err = errors.New("Update of primary key value is not allowed")
				return
			}
		}
	}
	return
}

// execute query against a DB, returns SQLUpdate. Detects RefID and builds rollback
func (qp queryProcessor) ExecuteQuery(sql string) (*structures.SQLUpdate, error) {
	qparsed, err := qp.ParseQuery(sql)

	if err != nil {
		return nil, err
	}
	return qp.ExecuteParsedQuery(qparsed)
}

// execute query from QueryParsed data.
func (qp queryProcessor) ExecuteParsedQuery(parsed QueryParsed) (*structures.SQLUpdate, error) {
	qp.Logger.Trace.Println("ExecuteParsedQuery " + string(parsed.SQL))
	su, err := qp.MakeSQLUpdateStructure(parsed)

	if err != nil {
		return nil, err
	}

	err = qp.DB.QM().ExecuteSQL(parsed.SQL)

	if err != nil {
		return nil, err
	}
	return &su, err
}

// Execute query from TX
func (qp queryProcessor) ExecuteQueryFromTX(sql structures.SQLUpdate) error {
	return qp.DB.QM().ExecuteSQL(string(sql.Query))
}

// Execute rollback query from TX
func (qp queryProcessor) ExecuteRollbackQueryFromTX(sql structures.SQLUpdate) error {
	return qp.DB.QM().ExecuteSQL(string(sql.RollbackQuery))
}

// Builds SQL update structure. It fins ID of a record, and build rollback query
func (qp queryProcessor) MakeSQLUpdateStructure(parsed QueryParsed) (sqlupdate structures.SQLUpdate, err error) {
	// get RefID info

	rollSQL, err := parsed.buildRollbackSQL()

	if err != nil {
		return
	}
	sqlupdate = structures.NewSQLUpdate(parsed.SQL, parsed.ReferenceID(), rollSQL)
	qp.Logger.Trace.Printf("rollback for %s is %s and refID %s", parsed.SQL, rollSQL, parsed.ReferenceID())
	return
}
