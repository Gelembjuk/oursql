package dbquery

import (
	"errors"
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
	r.SQL = sqlquery

	lcase := strings.ToLower(sqlquery)

	if strings.HasPrefix(lcase, "select ") {
		r.SQLKind = QueryKindSelect
	} else if strings.HasPrefix(lcase, "update ") {
		r.SQLKind = QueryKindUpdate
	} else if strings.HasPrefix(lcase, "insert ") {
		r.SQLKind = QueryKindUpdate
	} else if strings.HasPrefix(lcase, "delete ") {
		r.SQLKind = QueryKindUpdate
	}

	var err error
	r.Structure, err = qp.ParseQueryStructure(r.SQL)

	if err != nil {
		return r, err
	}
	return r, nil
}

// Parse query structure. gets table, ID, fields etc
func (qp queryProcessor) ParseQueryStructure(sqlquery string) (QueryStructure, error) {
	r := QueryStructure{}

	return r, nil
}

// checks if this query is syntax correct , return altered query if needed
func (qp queryProcessor) CheckQuerySyntax(sql string) error {
	return nil
}

// execute query against a DB
func (q queryProcessor) ExecteQuery(sql string) error {
	return q.DB.ExecuteSQL(sql)
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
