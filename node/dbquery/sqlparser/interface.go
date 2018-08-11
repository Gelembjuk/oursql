package sqlparser

const (
	QueryKindSelect = "select"
	QueryKindUpdate = "update"
	QueryKindInsert = "insert"
	QueryKindDelete = "delete"
	QueryKindCreate = "create"
	QueryKindDrop   = "drop"
	QueryKindOther  = "other"
)

type SQLQueryParserInterface interface {
	Parse(sqlquery string) error
	ExtendInsert(column string, value string, coltype string) error
	GetCanonicalQuery() string
	GetKind() string
	IsSingeTable() bool
	IsRead() bool
	IsModifyDB() bool
	GetTable() string
	IsTableManage() bool
	IsTableDataUpdate() bool
	GetUpdateColumns() map[string]string
	HasCondition() bool
	IsOneColumnCondition() bool
	GetOneColumnCondition() (string, string)
	GetComments() []string
}

func NewSqlParser() SQLQueryParserInterface {
	return &sqlParser{}
}
