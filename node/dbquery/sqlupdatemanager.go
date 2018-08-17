package dbquery

import (
	"bytes"
	"errors"

	"github.com/gelembjuk/oursql/lib"
	"github.com/gelembjuk/oursql/node/dbquery/sqlparser"
	"github.com/gelembjuk/oursql/node/structures"
)

// this are operations on structures.SQLUpdate where parsing is needed
// structures.SQLUpdate can not be moved here because it is part of transaction structure

type sqlUpdateManager struct {
	SQLUpdate structures.SQLUpdate
	Parsed    sqlparser.SQLQueryParserInterface
}

func (um *sqlUpdateManager) Init() error {
	um.Parsed = sqlparser.NewSqlParser()
	return um.Parsed.Parse(string(um.SQLUpdate.Query))
}

// Check if one SQL update can follow other update
// we allow:
// insert only after table create
// update only after insert or update
// delete only after insert or update
// nothing about delete or drop

func (um sqlUpdateManager) CheckUpdateCanFollow(sqlUpdPrev *structures.SQLUpdate) (err error) {
	// parse both queries
	var sqlparsed1 sqlparser.SQLQueryParserInterface

	sqlparsed1 = nil

	if sqlUpdPrev != nil {
		sqlparsed1 = sqlparser.NewSqlParser()

		err = sqlparsed1.Parse(string(sqlUpdPrev.Query))

		if err != nil {
			return
		}
	}

	if um.Parsed.GetKind() != lib.QueryKindCreate &&
		um.Parsed.GetKind() != lib.QueryKindInsert &&
		um.Parsed.GetKind() != lib.QueryKindUpdate &&
		um.Parsed.GetKind() != lib.QueryKindDelete {

		return errors.New("Operation is not an update query")
	}
	if sqlparsed1 == nil && um.Parsed.GetKind() != lib.QueryKindCreate {
		// only create can be based on nothing
		return errors.New("Operation is not allowed on base of given transaction")
	}

	if um.Parsed.GetKind() == lib.QueryKindCreate {
		// we always allow create of a table
		// NOTE this must be controlled additionally as part of a consensus
		return nil
	}

	// for all other operations a table of previous must be same as new operation
	if sqlparsed1.GetTable() != um.Parsed.GetTable() {
		return errors.New("Table of this SQL query must be same as a base transaction")
	}

	if um.Parsed.GetKind() == lib.QueryKindInsert {
		// only after create and on same table
		if sqlparsed1.GetKind() == lib.QueryKindCreate {
			// previous TX is a table create
			return
		}
	}

	if um.Parsed.GetKind() == lib.QueryKindUpdate ||
		um.Parsed.GetKind() == lib.QueryKindDelete {
		// only after create and on same table
		if sqlparsed1.GetKind() == lib.QueryKindInsert &&
			sqlparsed1.GetKind() == lib.QueryKindUpdate {
			// previous query was insert or update

			if bytes.Compare(sqlUpdPrev.ReferenceID, um.SQLUpdate.ReferenceID) == 0 &&
				len(sqlUpdPrev.ReferenceID) > 0 {
				// previous RefID must be same as now
				return
			}
		}

	}
	// in all other case we don't allow

	return errors.New("Operation is not allowed on base of given transaction")
}

// Detects if this SQL TX allows to have multiple following TXs
// this is possible if the query is table create
// or table create is based on empty transaction
func (um sqlUpdateManager) CheckAllowsMultipleSubtransactions(sqlUpdPrev *structures.SQLUpdate) (allow bool, err error) {
	if sqlUpdPrev == nil {
		if um.Parsed.GetKind() == lib.QueryKindCreate {
			allow = true
			return
		}

	}
	sqlparsed1 := sqlparser.NewSqlParser()

	err = sqlparsed1.Parse(string(sqlUpdPrev.Query))

	if err != nil {
		return
	}

	if sqlparsed1.GetKind() == lib.QueryKindCreate && um.Parsed.GetKind() == lib.QueryKindInsert {
		allow = true
		return
	}

	return
}
func (um sqlUpdateManager) GetAlternativeRefID() (RefID []byte, err error) {
	// if this is insert operation, return a table create RefID

	if um.Parsed.GetKind() == lib.QueryKindInsert {
		RefID = []byte(um.Parsed.GetTable() + ":*") // this is RefID of a table create  operation
		return
	}

	return nil, nil // no alternative refID
}

// Checks if a query requires base transactions
// This will be false only for a table create SQL query, true for any other
func (um sqlUpdateManager) RequiresBaseTransation() bool {
	if um.Parsed.GetKind() == lib.QueryKindCreate {
		return false
	}

	return true
}
