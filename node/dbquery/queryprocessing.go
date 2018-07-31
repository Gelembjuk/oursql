package dbquery

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
)
import b64 "encoding/base64"

type queryProcessor struct {
	DB     database.DBManager
	Logger *utils.LoggerMan
}

// checks if this query is syntax correct , return altered query if needed
func (qp queryProcessor) ParseQuery(sqlquery string) (QueryParsed, error) {
	sqlquery = strings.TrimSpace(sqlquery)

	r := QueryParsed{}

	// detect comments and remove them
	//re := regexp.MustCompile("\/\*.*\*\/")
	sqlquery, err := qp.parseQueryComments(sqlquery, &r)
	if err != nil {
		return r, err
	}

	err = qp.parseQueryKind(sqlquery, &r)
	if err != nil {
		return r, err
	}

	r.Structure, err = qp.parseQueryStructure(sqlquery)
	if err != nil {
		return r, err
	}

	err = qp.parseQueryReferenceID(sqlquery, &r)
	if err != nil {
		return r, err
	}

	r.SQL = sqlquery

	return r, nil
}

// Parse query structure. gets table, ID, fields etc
func (qp queryProcessor) parseQueryStructure(sqlquery string) (QueryStructure, error) {
	r := QueryStructure{}

	return r, nil
}

// Parse comments
func (qp queryProcessor) parseQueryComments(sqlquery string, r *QueryParsed) (updsql string, rerr error) {
	updsql = sqlquery

	return
}

// Parse kind
func (qp queryProcessor) parseQueryKind(sqlquery string, r *QueryParsed) error {
	lcase := strings.ToLower(sqlquery)

	if strings.HasPrefix(lcase, "select ") {
		r.SQLKind = database.QueryKindSelect
	} else if strings.HasPrefix(lcase, "update ") {
		r.SQLKind = database.QueryKindUpdate
	} else if strings.HasPrefix(lcase, "insert ") {
		r.SQLKind = database.QueryKindUpdate
	} else if strings.HasPrefix(lcase, "delete ") {
		r.SQLKind = database.QueryKindUpdate
	} else if strings.HasPrefix(lcase, "create table ") {
		r.SQLKind = database.QueryKindCreate
	} else if strings.HasPrefix(lcase, "drop table ") {
		r.SQLKind = database.QueryKindDrop
	} else {
		return errors.New("Unknown query type")
	}

	return nil
}

// get reference ID from the query. look on condition
func (qp queryProcessor) parseQueryReferenceID(sqlquery string, r *QueryParsed) error {
	if r.SQLKind == database.QueryKindSelect {
		return nil
	}
	return nil
}

// checks if this query is syntax correct , return altered query if needed
func (qp queryProcessor) CheckQuerySyntax(sql string) error {
	return nil
}

// execute query against a DB, returns SQLUpdate. Detects RefID and builds rollback
func (q queryProcessor) ExecuteQuery(sql string) (*structures.SQLUpdate, error) {
	qp, err := q.ParseQuery(sql)
	if err != nil {
		return nil, err
	}
	return q.ExecuteParsedQuery(qp)
}

// execute query from QueryParsed data.
func (q queryProcessor) ExecuteParsedQuery(qp QueryParsed) (*structures.SQLUpdate, error) {
	// must get refID
	id, err := q.DB.QM().ExecuteSQLFirstly(qp.SQL, qp.SQLKind)

	if err != nil {
		return nil, err
	}
	su := &structures.SQLUpdate{}
	su.Query = []byte(qp.SQL)

	if qp.SQLKind == database.QueryKindInsert {
		// refID is TABLE:KEYVAL
		su.ReferenceID = []byte(qp.Structure.Table)
		su.ReferenceID = append(su.ReferenceID, []byte(":")...)
		su.ReferenceID = append(su.ReferenceID, []byte(strconv.FormatInt(id, 10))...)
	} else if qp.SQLKind == database.QueryKindCreate {
		su.ReferenceID = []byte(qp.Structure.Table)
		su.ReferenceID = append(su.ReferenceID, []byte(":*")...)
	} else {
		// in other cases we already must have a refID
		su.ReferenceID = []byte(qp.ReferenceID)
	}
	return su, nil
}

// Execute query from TX
func (q queryProcessor) ExecuteQueryFromTX(sql structures.SQLUpdate) error {
	return q.DB.QM().ExecuteSQL(string(sql.Query))
}

// Execute rollback query from TX
func (q queryProcessor) ExecuteRollbackQueryFromTX(sql structures.SQLUpdate) error {
	return q.DB.QM().ExecuteSQL(string(sql.RollbackQuery))
}

// errorKind possible values: 2 - pubkey required, 3 - data sign required
func (q queryProcessor) FormatSpecialErrorMessage(errorKind uint, txdata []byte, datatosign []byte) (string, error) {
	if errorKind == 2 {
		return "Error(2): Public Key required", nil
	}
	if errorKind == 3 {
		txdataB64 := b64.StdEncoding.EncodeToString([]byte(txdata))
		datatosignB64 := b64.StdEncoding.EncodeToString([]byte(datatosign))
		return "Error(3): Signature required:" + txdataB64 + "::" + datatosignB64, nil
	}
	return "", errors.New("Unknown error kind")
}

// Builds SQL update structure. It fins ID of a record, and build rollback query
func (q queryProcessor) MakeSQLUpdateStructure(sql string) (structures.SQLUpdate, error) {
	refID := "aaa"
	rollbackSQL := "BBB"
	s := structures.NewSQLUpdate(sql, refID, rollbackSQL)

	return s, nil
}
