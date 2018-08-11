package sqlparser

import (
	"reflect"
	"strconv"
	"testing"
)

func TestBasic(t *testing.T) {
	p := NewSqlParser()
	sqls := map[string][]string{
		"UPDATE t SET a='b' WHERE id=1":                                                       []string{"UPDATE t SET a='b' WHERE id=1", "update", "t", "1"},
		" UPDATE t SET a='b' WHERE id='1';":                                                   []string{"UPDATE t SET a='b' WHERE id='1'", "update", "t", "1"},
		" UPDATE t SET a='b' WHERE id='1' ; ":                                                 []string{"UPDATE t SET a='b' WHERE id='1'", "update", "t", "1"},
		" UPDATE t SET a='b',c = 'X', d = 2, `e`= \"333\" WHERE id='1' ":                      []string{"UPDATE t SET a='b',c = 'X', d = 2, `e`= \"333\" WHERE id='1'", "update", "t", "4"},
		" UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1' /*SIGN:XXXXXX;DATA:YYYYYYYYYYYYY;*/": []string{"UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1'", "update", "t", "3"},
		"UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1' /*PUBKEY:ZZZZZZZZ;*/":                 []string{"UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1'", "update", "t", "3"},
		"INSERT into t SET a='b',c = 'X', d = 2  /*PUBKEY:ZZZZZZZZ;*/":                        []string{"INSERT into t SET a='b',c = 'X', d = 2", "insert", "t", "3"},
		"INSERT into t (a,`b`, c) values (2,'oooo\\\"', \"123 \\n\" )":                        []string{"INSERT into t (a,`b`, c) values (2,'oooo\\\"', \"123 \\n\" )", "insert", "t", "3"},
		"select * from aaa":                                                                   []string{"select * from aaa", "select", "aaa", "0"},
		"delete FROM t WHERE x=y":                                                             []string{"delete FROM t WHERE x=y", "delete", "t", "0"},
		"create table ttt (a int, b varchar(10))":                                             []string{"create table ttt (a int, b varchar(10))", "create", "ttt", "0"},
		" drop table ttt;":                                                                    []string{"drop table ttt", "drop", "ttt", "0"},
		" UPDATE t SET a='b',c = 'X\\\"q', `d` = 2, `e`= \"3\\'33\",p = `oo\\r`": []string{"UPDATE t SET a='b',c = 'X\\\"q', `d` = 2, `e`= \"3\\'33\",p = `oo\\r`", "update", "t", "5"}}

	for sql, res := range sqls {
		err := p.Parse(sql)

		if err != nil {
			t.Fatalf("Error: %s for %s", err.Error(), sql)
		}

		if res[0] != p.GetCanonicalQuery() {
			t.Fatalf("Canonical different: %s vs %s", p.GetCanonicalQuery(), res[0])
		}

		if res[1] != p.GetKind() {
			t.Fatalf("Kind different: %s vs %s for %s", p.GetKind(), res[1], sql)
		}

		if res[2] != p.GetTable() {
			t.Fatalf("Table different: %s vs %s for %s", p.GetTable(), res[2], sql)
		}

		if res[3] != strconv.Itoa(len(p.GetUpdateColumns())) {
			t.Fatalf("Number of update columns is different: %s vs %d for %s", res[3], len(p.GetUpdateColumns()), sql)
		}
	}
}

func TestComments(t *testing.T) {
	p := NewSqlParser()
	sqls := map[string][]string{
		" UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1' ":                                               []string{},
		" UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1' /*SIGN:XXXXXX;DATA:YYYYYYYYYYYYY;*/":            []string{"SIGN:XXXXXX;DATA:YYYYYYYYYYYYY;"},
		" UPDATE t SET a='b',c = 'X',/* other */ d = 2 WHERE id='1' /*SIGN:XXXXXX;DATA:YYYYYYYYYYYYY;*/": []string{" other ", "SIGN:XXXXXX;DATA:YYYYYYYYYYYYY;"},
		"UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1' /*PUBKEY:ZZZZZZZZ;*/":                            []string{"PUBKEY:ZZZZZZZZ;"}}
	for sql, res := range sqls {
		err := p.Parse(sql)

		if err != nil {
			t.Fatal(err.Error())
		}

		comments := p.GetComments()

		if !reflect.DeepEqual(comments, res) {
			t.Fatalf("Comments different: %s vs %s", comments, res)
		}
	}
}

func TestUpdateColumns(t *testing.T) {
	p := NewSqlParser()

	sqls := map[string]map[string]string{
		"UPDATE t SET a='b' WHERE id=1":                                  map[string]string{"a": "b"},
		" UPDATE t SET a='b',c = 'X', d = 2, `e`= \"333\" WHERE id='1' ": map[string]string{"a": "b", "c": "X", "d": "2", "e": "333"},
		" UPDATE t SET a='b',c = 'X', d = 2 WHERE id='1'":                map[string]string{"a": "b", "c": "X", "d": "2"},
		"INSERT into t SET a='b',c = 'X', d = 2  /*PUBKEY:ZZZZZZZZ;*/":   map[string]string{"a": "b", "c": "X", "d": "2"},
		"INSERT into t (a,`b`, c) values (2,'oooo\\\"', \"123 \\n\" )":   map[string]string{"a": "2", "b": "oooo\"", "c": "123 \n"},
		"select * from aaa":                                              map[string]string{},
		"delete FROM t WHERE x=y":                                        map[string]string{},
		"create table ttt (a int, b varchar(10))":                        map[string]string{},
		" drop table ttt;":                                               map[string]string{},
		" UPDATE t SET a='b',c = 'X\\\"q', `d` = 2, `e`= \"3\\'33\",p = `oo\\r`": map[string]string{"a": "b", "c": "X\"q", "d": "2", "e": "3'33", "p": "oo\r"}}

	for sql, res := range sqls {
		err := p.Parse(sql)

		if err != nil {
			t.Fatalf("Error: %s for %s", err.Error(), sql)
		}

		if !reflect.DeepEqual(res, p.GetUpdateColumns()) {
			t.Fatalf("Fail for: %s : expected: %s , got: %s", sql, res, p.GetUpdateColumns())
		}

	}
}
func TestCondition(t *testing.T) {
	p := NewSqlParser()
	sqls := map[string][]string{
		" UPDATE t SET a='b',c = 'X\\\"q', `d` = 2":                                                      []string{"", ""},
		" UPDATE t SET a='b',c = 'X\\\"q', `d` = 2 WHERE id='2' and y='x'":                               []string{"", ""},
		" UPDATE t SET a='b',c = 'X\\\"q', `d` = 2 WHERE id='2' and y = 'x' OR z=\"bbb\"":                []string{"", ""},
		" UPDATE t SET a='b',c = 'X\\\"q', `d` = 2 WHERE id='2' and y='x' OR z=\"bbb\" AND (x=1 or y=2)": []string{"", ""},
		"UPDATE t SET a='b' WHERE id='2' GROUP by x order BY z LIMIT  10,0":                              []string{"id", "2"},
		"UPDATE t SET a='b' WHERE id='2' order BY z, b LIMIT  10,0":                                      []string{"id", "2"},
		"UPDATE t SET a='b' WHERE id='2' LIMIT  10,0":                                                    []string{"id", "2"},
		" delete from  t WHERE id = '2' LIMIT  10,0":                                                     []string{"id", "2"},
		" delete from  t WHERE id = '2' ":                                                                []string{"id", "2"},
		" delete from  t WHERE id <> '2' ":                                                               []string{"", ""}}

	for sql, res := range sqls {
		err := p.Parse(sql)

		if err != nil {
			t.Fatalf("Error: %s for %s", err.Error(), sql)
		}

		x, y := p.GetOneColumnCondition()

		if res[0] != x || res[1] != y {
			t.Fatalf("Fail for: %s : expected: %s,%s , got: %s,%s", sql, res[0], res[1], x, y)
		}

	}
}
func TestConditionString(t *testing.T) {
	p := &sqlParser{}

	tests := map[string]map[string][]string{
		"a='b' AND c = 'X\\\"q' OR `d` = 2": map[string][]string{"a": []string{"b", "="}, "c": []string{"X\"q", "="}, "d": []string{"2", "="}},
		"a <> 'b'":                          map[string][]string{"a": []string{"b", "<>"}},
		"id='2' and y='x'":                  map[string][]string{"id": []string{"2", "="}, "y": []string{"x", "="}},

		"id='2' and y!='x' and z = 2 OR p= 3 OR p <>3":                      map[string][]string{"id": []string{"2", "="}, "y": []string{"x", "!="}, "z": []string{"2", "="}, "p": []string{"3", "<>"}},
		"id='2' and y = 'x' OR z=\"bbb\"":                                   map[string][]string{"id": []string{"2", "="}, "y": []string{"x", "="}, "z": []string{"bbb", "="}},
		"id='2' and y='x' OR z=\"bbb\" AND (x=1 or y=2)":                    map[string][]string{"id": []string{"2", "="}, "y": []string{"x", "="}, "z": []string{"bbb", "="}},
		"id='2' and y='x' OR z=\"bbb\" AND (x=1 or y=2 and (a=1 or b = 2))": map[string][]string{"id": []string{"2", "="}, "y": []string{"x", "="}, "z": []string{"bbb", "="}}}
	// TODO . this test fails. Query parser needs improvement
	//"id='2' and y='tt\\\"oo\\''":                                        map[string][]string{"id": []string{"2", "="}, "y": []string{"tt\"oo'", "="}},

	for s, res := range tests {
		data, err := p.parseConditionString(s)

		if err != nil {
			t.Fatalf("Error: %s for %s", err.Error(), s)
		}

		if !reflect.DeepEqual(res, data) {
			t.Fatalf("Fail for: %s : expected: %s , got: %s", s, res, data)
		}

	}
}

func TestInsert(t *testing.T) {
	p := NewSqlParser()
	sqls := map[string][]string{
		"INSERT into t SET a='b',c = 'X', kc = 2":                      []string{"INSERT into t SET a='b',c = 'X', kc = 2"},
		"INSERT into t SET a='b',c = 'X', d = 2  /*PUBKEY:ZZZZZZZZ;*/": []string{"INSERT into t SET a='b',c = 'X', d = 2, kc='5'"},
		"INSERT into t (a,`b`, c) values (2,'oooo\\\"', \"123 \\n\" )": []string{"INSERT into t (kc, a,`b`, c) values ('5', 2,'oooo\\\"', \"123 \\n\" )"}}

	for sql, res := range sqls {
		err := p.Parse(sql)

		if err != nil {
			t.Fatalf("Error: %s for %s", err.Error(), sql)
		}

		err = p.ExtendInsert("kc", "5", "string")

		if err != nil {
			t.Fatalf("Error Processing: %s for %s", err.Error(), sql)
		}
		if res[0] != p.GetCanonicalQuery() {
			t.Fatalf("Canonical different: %s vs %s", p.GetCanonicalQuery(), res[0])
		}

	}
}
