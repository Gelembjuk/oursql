package dbquery

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
)

func TestParsingBasic(t *testing.T) {
	DBM := database.GetDBManagerMock()
	si := database.SQLExplainInfo{}
	si.SelectType = "UPDATE"
	si.Table = "t"
	DBM.SetSQLExplain(&si)

	sqls := map[string]QueryParsed{
		"UPDATE t SET a='b' WHERE id=1":                                                       QueryParsed{"UPDATE t SET a='b' WHERE id=1", "update", "t:1", []byte{}, []byte{}, []byte{}, QueryStructure{}},
		" UPDATE t SET a='b' WHERE id='1';":                                                   QueryParsed{"UPDATE t SET a='b' WHERE id='1'", "update", "t:1", []byte{}, []byte{}, []byte{}, QueryStructure{}},
		" UPDATE t SET a='b' WHERE id='1' ; ":                                                 QueryParsed{"UPDATE t SET a='b' WHERE id='1'", "update", "t:1", []byte{}, []byte{}, []byte{}, QueryStructure{}},
		" UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1' ":                                    QueryParsed{"UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1'", "update", "t:1", []byte{}, []byte{}, []byte{}, QueryStructure{}},
		" UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1' /*SIGN:XXXXXX;DATA:YYYYYYYYYYYYY;*/": QueryParsed{"UPDATE t SET a='b' WHERE id='1'", "update", "t:1", []byte{}, []byte("XXXXXX"), []byte("YYYYYYYYYYYYY"), QueryStructure{}},
		"UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1' /*PUBKEY:ZZZZZZZZ;*/":                 QueryParsed{"UPDATE t SET a='b' WHERE id='1'", "update", "t:1", []byte("ZZZZZZZZ"), []byte{}, []byte{}, QueryStructure{}}}

	qp := NewQueryProcessor(&DBM, utils.CreateLoggerStdout())

	for sql, res := range sqls {
		parsed, err := qp.ParseQuery(sql)

		if err != nil {
			t.Fatalf("Parse error: %s", err.Error())
		}
		if !reflect.DeepEqual(parsed, res) {
			fmt.Println(parsed)
			fmt.Println(res)
			t.Fatalf("Different from expected for: %s", sql)
		}

	}

}
