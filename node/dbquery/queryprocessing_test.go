package dbquery

import (
	"testing"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
)

func TestParsingUpdate(t *testing.T) {
	DBM := database.GetDBManagerMock()
	si := database.SQLExplainInfo{}
	si.SelectType = "UPDATE"
	si.Table = "t"
	DBM.SetSQLExplain(&si)

	DBM.KeyColumn = "id"

	sqls := map[string][]string{
		"UPDATE t SET a='b' WHERE id=1":                []string{"UPDATE t SET a='b' WHERE id=1", "update", "t:1", "", "", ""},
		" UPDATE t SET a='b' WHERE id='1';":            []string{"UPDATE t SET a='b' WHERE id='1'", "update", "t:1", "", "", ""},
		" UPDATE t SET a='b' WHERE id = 'tt\\\"oo' ; ": []string{"UPDATE t SET a='b' WHERE id = 'tt\\\"oo'", "update", "t:tt\"oo", "", "", ""},
		//" UPDATE t SET a='b' WHERE id = 'tt\\\"oo\\'' ; ":                                     []string{"UPDATE t SET a='b' WHERE id = 'tt\\\"oo\\''", "update", "t:tt\"oo'", "", "", ""},
		" UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1' ":                                    []string{"UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1'", "update", "t:1", "", "", ""},
		" UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1' /*SIGN:XXXXXX;DATA:YYYYYYYYYYYYY;*/": []string{"UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1'", "update", "t:1", "", "", ""},
		"UPDATE t SET a='b',c = 'X', d = 2 WHERE id='2' /*PUBKEY:ZZZZZZZZ;*/":                 []string{"UPDATE t SET a='b',c = 'X', d = 2 WHERE id='2'", "update", "t:2", "", "", ""}}

	qp := NewQueryProcessor(&DBM, utils.CreateLoggerStdout())

	for sql, res := range sqls {
		parsed, err := qp.ParseQuery(sql)

		if err != nil {
			t.Fatalf("Parse error: %s for %s", err.Error(), sql)
		}

		if parsed.SQL != res[0] {
			t.Fatalf("SQL different: got: %s, expected: %s", sql, res[0])
		}
		if parsed.Structure.GetKind() != res[1] {
			t.Fatalf("SQL kind different: got: %s, expected: %s for %s", parsed.Structure.GetKind(), res[0], sql)
		}
		if !parsed.IsUpdate() {
			t.Fatalf("SQL should be update type: %s", sql)
		}

		if parsed.ReferenceID() != res[2] {
			t.Fatalf("RefID different: got: %s, expected: %s for %s", string(parsed.ReferenceID()), res[2], sql)
		}
	}

}
