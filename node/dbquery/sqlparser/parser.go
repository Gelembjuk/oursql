package sqlparser

import (
	"errors"
	"regexp"
	"strings"

	"github.com/gelembjuk/oursql/lib"
	"github.com/gelembjuk/oursql/node/database"
)

const (
	querySubKindInsertSet    = "insertset"
	querySubKindInsertValues = "insertvalues"
)

type sqlParser struct {
	originalQuery    string
	canonicalQuery   string
	kind             string
	subkind          string
	table            string
	comments         []string
	updateColumns    map[string]string
	conditonText     string
	conditionColumns map[string][]string
}

func (q *sqlParser) Parse(sqlquery string) (err error) {
	q.originalQuery = sqlquery
	q.canonicalQuery = ""
	q.kind = ""
	q.subkind = ""
	q.table = ""
	q.comments = []string{}
	q.updateColumns = map[string]string{}
	q.conditonText = ""
	q.conditionColumns = map[string][]string{}

	sqlquery = strings.TrimSpace(sqlquery)
	sqlquery, comments, err := q.parseComments(sqlquery)

	if err != nil {
		return
	}

	q.comments = comments

	sqlquery, _ = q.normalizeQuery(sqlquery)

	q.canonicalQuery = sqlquery

	q.kind, q.table, err = q.parseKindAndTable(sqlquery)

	if err != nil {
		return
	}

	q.updateColumns, err = q.parseUpdateColumns(sqlquery, q.kind)

	if err != nil {
		return
	}
	q.conditonText, q.conditionColumns, err = q.parseCondition(sqlquery, q.kind)

	if err != nil {
		return
	}

	return
}

// updates already parsed query if it is insert
// adds one more column to a query
func (q *sqlParser) ExtendInsert(column string, value string, coltype string) error {
	if q.GetKind() != lib.QueryKindInsert {
		return errors.New("Now insert query")
	}

	// check if column is already in the list of columns
	if _, ok := q.updateColumns[column]; ok {
		// column already is present in a condition
		return nil
	}

	if q.subkind == querySubKindInsertSet {
		extraCond := column + "="

		if coltype == "int" {
			extraCond = extraCond + database.Quote(value)
		} else {
			extraCond = extraCond + "'" + database.Quote(value) + "'"
		}
		q.canonicalQuery = q.canonicalQuery + ", " + extraCond

		return nil

	}
	if q.subkind == querySubKindInsertValues {
		var extraCond string
		if coltype == "int" {
			extraCond = database.Quote(value)
		} else {
			extraCond = "'" + database.Quote(value) + "'"
		}
		// insert as a first column
		re, err := regexp.Compile("(?i)^(.+into\\s+" + q.GetTable() + "\\s+\\()(.+\\)\\s+values\\s+\\()(.+)$")

		if err != nil {
			return err
		}
		s := re.FindStringSubmatch(q.canonicalQuery)

		if len(s) < 3 {
			return errors.New("Can not parse INSERT query")
		}
		q.canonicalQuery = s[1] + column + ", " + s[2] + extraCond + ", " + s[3]

		return nil
	}

	return errors.New("Unknown query type")
}

// ================== PARSERS =============================
// extract comments from the query
func (q *sqlParser) parseComments(originalsqlquery string) (sqlquery string, comments []string, err error) {
	comments = []string{}
	sqlquery = originalsqlquery

	r, err := regexp.Compile("/\\*(.*?)\\*/")

	if err != nil {
		return
	}

	fs := r.FindAllStringSubmatch(sqlquery, -1)

	for _, s := range fs {
		comment := s[1]
		sqlquery = strings.Replace(sqlquery, s[0], " ", 1)
		comments = append(comments, comment)
	}

	return
}

// normalize a query. remove extra spaces (but not from texts inside)
// TODO
// this muts do similar work like "github.com/percona/go-mysql/query"
// but without replacing values with ?
func (q *sqlParser) normalizeQuery(sqlquery string) (string, error) {
	sqlquery = strings.Trim(sqlquery, ";")
	sqlquery = strings.TrimSpace(sqlquery)
	return sqlquery, nil
}

// get query kind and table name
func (q *sqlParser) parseKindAndTable(sqlquery string) (kind string, table string, err error) {
	lcase := strings.ToLower(sqlquery)

	re := ""

	if strings.HasPrefix(lcase, "select ") {
		kind = lib.QueryKindSelect
		re = "from\\s+([^ ]+)"

	} else if strings.HasPrefix(lcase, "show ") {
		kind = lib.QueryKindSelect
		re = ""

	} else if strings.HasPrefix(lcase, "update ") {
		kind = lib.QueryKindUpdate
		re = "update\\s+([^ ]+)\\s"

	} else if strings.HasPrefix(lcase, "insert ") {
		kind = lib.QueryKindInsert
		re = "insert\\s+into\\s+([^ ]+)\\s"

	} else if strings.HasPrefix(lcase, "delete ") {
		kind = lib.QueryKindDelete
		re = "delete\\s+from\\s+([^ ]+)\\s"

	} else if strings.HasPrefix(lcase, "create table ") {
		kind = lib.QueryKindCreate
		re = "create\\s+table\\s+([^ ]+)\\s"

	} else if strings.HasPrefix(lcase, "drop table ") {
		kind = lib.QueryKindDrop
		re = "drop\\s+table\\s+([^ ]+)"

	} else if strings.HasPrefix(lcase, "set ") {
		kind = lib.QueryKindSet
		re = ""

	} else {
		err = errors.New("Unknown query type")
	}

	if re != "" {
		var r *regexp.Regexp
		r, err = regexp.Compile(re)

		if err != nil {
			return
		}
		sr := r.FindStringSubmatch(lcase)

		if len(sr) < 2 {
			err = errors.New("Table name not found")
			return
		}

		table = sr[1]
	}

	return
}

// parse update columns and values
func (q *sqlParser) parseUpdateColumns(sqlquery string, kind string) (data map[string]string, err error) {
	data = map[string]string{}

	if kind != lib.QueryKindInsert && kind != lib.QueryKindUpdate {
		return
	}

	var r *regexp.Regexp

	if kind == lib.QueryKindInsert {
		r, err = regexp.Compile("(?i)insert\\s+into\\s+[^ ]+\\s+set\\s+(.+)")

		if err != nil {
			return
		}
		sr := r.FindStringSubmatch(sqlquery)

		if len(sr) >= 2 {
			q.subkind = querySubKindInsertSet
			return q.parseKeyValueSet(sr[1])
		}

		r, err = regexp.Compile("(?i)insert\\s+into\\s+[^ ]+\\s+\\((.*?)\\)\\s+values\\s+\\((.+)\\)")

		if err != nil {
			return
		}

		sr = r.FindStringSubmatch(sqlquery)

		if len(sr) >= 3 {
			q.subkind = querySubKindInsertValues
			return q.parseValueList(sr[1], sr[2])
		}

		err = errors.New("Can not parse keys/values from INSERT query")
	} else if kind == lib.QueryKindUpdate {

		r, err = regexp.Compile("(?i)update\\s+[^ ]+\\s+set\\s+(.+)\\swhere")

		if err != nil {
			return
		}
		sr := r.FindStringSubmatch(sqlquery)

		if len(sr) >= 2 {
			return q.parseKeyValueSet(sr[1])
		}

		r, err = regexp.Compile("(?i)update\\s+[^ ]+\\s+set\\s+(.+)")

		if err != nil {
			return
		}

		sr = r.FindStringSubmatch(sqlquery)

		if len(sr) >= 2 {
			return q.parseKeyValueSet(sr[1])
		}

		err = errors.New("Can not parse keys/values from UPDATE query")

	}

	return
}

// parse update columns and values
func (q *sqlParser) parseKeyValueSet(command string) (map[string]string, error) {
	data := map[string]string{}
	r, err := regexp.Compile("((?:[^,\"']|\"[^\"]*\"|'[^']*')+)")

	if err != nil {
		return nil, err
	}

	st := r.FindAllString(command, -1)

	for _, s := range st {
		s = strings.TrimSpace(s)
		kv := strings.SplitN(s, "=", 2)

		if len(kv) != 2 {
			continue
		}
		k := q.cleanSQLColumnName(kv[0])
		v := q.cleanSQLValue(kv[1])

		data[k] = v

	}

	return data, nil
}

// parse names and values from insert lists
func (q *sqlParser) parseValueList(nameslist, valueslist string) (map[string]string, error) {
	data := map[string]string{}

	nameslist = strings.TrimSpace(nameslist)
	valueslist = strings.TrimSpace(valueslist)

	r, err := regexp.Compile("((?:[^,\"']|\"[^\"]*\"|'[^']*')+)")

	if err != nil {
		return nil, err
	}

	names := []string{}
	values := []string{}

	st := r.FindAllString(nameslist, -1)

	for _, s := range st {
		s = q.cleanSQLColumnName(s)
		names = append(names, s)
	}
	st = r.FindAllString(valueslist, -1)

	for _, s := range st {
		s = q.cleanSQLValue(s)
		values = append(values, s)
	}

	if len(names) != len(values) {
		return nil, errors.New("Can not parse names/values. Counts in lists are different")
	}

	for i, k := range names {
		data[k] = values[i]
	}

	return data, nil
}

// parse condition
func (q *sqlParser) parseCondition(sqlquery string, kind string) (condtext string, columns map[string][]string, err error) {
	columns = map[string][]string{}

	if kind != lib.QueryKindDelete && kind != lib.QueryKindUpdate {
		return
	}

	var r *regexp.Regexp

	if kind == lib.QueryKindDelete {
		r, err = regexp.Compile("(?i)delete\\s+from\\s+[^ ]+\\s+where\\s+(.+)")

		if err != nil {
			return
		}
		sr := r.FindStringSubmatch(sqlquery)

		if len(sr) >= 2 {
			return q.parseConditionDetails(sr[1])
		}

	} else if kind == lib.QueryKindUpdate {

		r, err = regexp.Compile("(?i)update\\s+[^ ]+\\s+set\\s+.+\\swhere\\s+(.+)")

		if err != nil {
			return
		}
		sr := r.FindStringSubmatch(sqlquery)

		if len(sr) >= 2 {
			return q.parseConditionDetails(sr[1])
		}

	}

	return
}

// parse condition details
func (q *sqlParser) parseConditionDetails(conditionstring string) (condtext string, columns map[string][]string, err error) {
	columns = map[string][]string{}
	conditionstring = strings.TrimSpace(conditionstring)
	// remove LIMIT, ORDER BY , GROUP BY from the end
	var r *regexp.Regexp

	r, err = regexp.Compile("(?i)^(.+)\\slimit\\s+\\d+[^\"']+$")

	if err != nil {
		return
	}
	sr := r.FindStringSubmatch(conditionstring)

	if len(sr) >= 2 {
		conditionstring = sr[1]
	}

	r, err = regexp.Compile("(?i)^(.+)\\sorder\\s+by\\s+[^\"']+$")

	if err != nil {
		return
	}
	sr = r.FindStringSubmatch(conditionstring)

	if len(sr) >= 2 {
		conditionstring = sr[1]
	}

	r, err = regexp.Compile("(?i)^(.+)\\sgroup\\s+by\\s+[^\"']+$")

	if err != nil {
		return
	}
	sr = r.FindStringSubmatch(conditionstring)

	if len(sr) >= 2 {
		conditionstring = sr[1]
	}
	conditionstring = strings.TrimSpace(conditionstring)

	// remove extra brekets
	if strings.HasPrefix(conditionstring, "(") {
		conditionstring = strings.TrimLeft(conditionstring, "(")
		conditionstring = strings.TrimRight(conditionstring, ")")
	}
	condtext = conditionstring
	columns, err = q.parseConditionString(condtext)

	return
}

// parse condition details to columns.
// NOTE we don't care about logic if there are AND,OR,NOT . We don't need it at this place
// NOTE this function can not parse all condition operators, only basic >=|<=|<>|!=|=|>|<
// all operators aka: LIKE, NOT LIKE, IS, NOT IS are not recognized here. maybe in the future
func (q *sqlParser) parseConditionString(conditionstring string) (columns map[string][]string, err error) {
	columns = map[string][]string{}

	var r *regexp.Regexp

	r, err = regexp.Compile("((?:[^ \"']|\"[^\"]*\"|'[^']*')+)")

	if err != nil {
		return
	}

	operators := ">=|<=|<>|!=|=|>|<"

	rCOV, err := regexp.Compile("^`?([a-zA-Z0-9_]+)`?(" + operators + ")(.+)$")

	if err != nil {
		return
	}

	rCO, err := regexp.Compile("^`?([a-zA-Z0-9_]+)`?(" + operators + ")$")

	if err != nil {
		return
	}

	rOV, err := regexp.Compile("^(" + operators + ")(.+)$")

	if err != nil {
		return
	}
	rO, err := regexp.Compile("^(" + operators + ")$")

	if err != nil {
		return
	}

	st := r.FindAllString(conditionstring, -1)

	column := ""
	operator := ""
	complicated := 0

	for _, s := range st {

		sl := strings.ToLower(s)

		if sl == "and" || sl == "or" || sl == "not" {
			continue
		}
		// s can be column name or value or operator or all together

		if column == "" {
			if strings.HasPrefix(s, "(") {
				complicated = complicated + 1
				// this is special case. we don't parse this yet.
				continue
			}
			if complicated > 0 {
				if strings.HasSuffix(s, ")") {
					complicated = complicated - 1
				}
				continue
			}
			// test for ColumnOpValue, aka xxx!='ccc'
			ss := rCOV.FindStringSubmatch(s)

			if len(ss) > 2 {
				columns[ss[1]] = []string{q.cleanSQLValue(ss[3]), ss[2]} // (VAUE, OPERATOR)
				continue
			}
			// test for ColumnOp aka xxx<>
			ss = rCO.FindStringSubmatch(s)

			if len(ss) >= 2 {
				column = ss[1]
				operator = ss[2]
				continue
			}

			column = q.cleanSQLColumnName(s)
		} else if operator == "" {
			// this is can be just operator. but we have to verify it is really operator
			ss := rO.FindStringSubmatch(s)

			if len(ss) >= 2 {
				operator = ss[1]
				continue
			}
			// test for ColumnOpValue, aka >20
			ss = rOV.FindStringSubmatch(s)

			if len(ss) >= 2 {
				columns[column] = []string{q.cleanSQLValue(ss[2]), ss[1]} // (VAUE, OPERATOR)
				column = ""
				continue
			}

			// ELSE this is unknown operator. we can not parse this. we start all from scratch to find next "normal" pair
			column = ""
		} else {
			// this is just a value
			value := q.cleanSQLValue(s)
			columns[column] = []string{value, operator}
			column = ""
			operator = ""
		}

	}

	return
}

// deescape value , remove quotes. returns only value that is set for  column
func (q *sqlParser) cleanSQLValue(value string) string {
	value = strings.TrimSpace(value)

	deescape := true

	if strings.HasPrefix(value, "\"") {
		value = strings.Trim(value, "\"")
	} else if strings.HasPrefix(value, "'") {
		value = strings.Trim(value, "'")
	} else if strings.HasPrefix(value, "`") {
		value = strings.Trim(value, "`")
	} else {
		deescape = false
	}

	if deescape {
		replace := map[string]string{"\\": "\\\\", "'": `\'`, "\\0": "\\\\0", "\n": "\\n", "\r": "\\r", `"`: `\"`, "\x1a": "\\Z"}

		for b, a := range replace {
			value = strings.Replace(value, a, b, -1)
		}
	}

	return value
}

// Clean column name. remove quotes, trim spaces etc
func (q *sqlParser) cleanSQLColumnName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "`")
	return name
}

// ================== END PARSERS =============================
func (q sqlParser) GetCanonicalQuery() string {
	return q.canonicalQuery
}
func (q sqlParser) GetKind() string {
	return q.kind
}
func (q sqlParser) IsSingeTable() bool {
	return q.kind != QueryKindOther
}
func (q sqlParser) IsRead() bool {
	return q.kind == QueryKindOther || q.kind == QueryKindSelect
}
func (q sqlParser) IsModifyDB() bool {
	return !(q.kind == QueryKindOther || q.kind == QueryKindSelect)
}
func (q sqlParser) GetTable() string {
	return q.table

}
func (q sqlParser) IsTableManage() bool {
	return q.kind == QueryKindDrop || q.kind == QueryKindCreate
}
func (q sqlParser) IsTableDataUpdate() bool {
	return q.kind == QueryKindDelete || q.kind == QueryKindInsert || q.kind == QueryKindUpdate
}
func (q sqlParser) GetUpdateColumns() map[string]string {
	return q.updateColumns
}
func (q sqlParser) HasCondition() bool {
	return len(q.conditonText) > 0
}
func (q sqlParser) IsOneColumnCondition() bool {
	if len(q.conditionColumns) != 1 {
		return false
	}
	for _, v := range q.conditionColumns {
		return v[1] == "="
	}
	return false
}

// this returns data only if there is single condition and it is "=" operator
func (q sqlParser) GetOneColumnCondition() (string, string) {
	// if there are more 1 conditions, we don't know which to return. ordering is unknown, so it can be different each time
	if len(q.conditionColumns) != 1 {
		return "", ""
	}

	for k, v := range q.conditionColumns {
		if v[1] == "=" {
			return k, v[0]
		}
		break
	}
	return "", ""
}
func (q sqlParser) GetComments() []string {
	return q.comments
}
