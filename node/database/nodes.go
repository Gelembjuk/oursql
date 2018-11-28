package database

const nodesTable = "nodes"

type Nodes struct {
	DB          *MySQLDB
	tablePrefix string
	tableName   string
}

func (ns *Nodes) getTableName() string {
	if ns.tableName == "" {
		ns.tableName = ns.DB.tablesPrefix + nodesTable
	}
	return ns.tableName
}

// Init new DB. Create table.
func (ns *Nodes) InitDB() error {
	return ns.DB.CreateTable(ns.getTableName(), "VARBINARY(100)", "VARBINARY(200)")
}

// retrns nodes list iterator
func (ns *Nodes) ForEach(callback ForEachKeyIteratorInterface) error {
	return ns.DB.forEachInTable(ns.getTableName(), callback)
}

// get count of records in the table
func (ns *Nodes) GetCount() (int, error) {
	return ns.DB.getCountInTable(ns.getTableName())
}

// Save node info
func (ns *Nodes) PutNode(nodeID []byte, nodeData []byte) error {
	err := ns.DB.Delete(ns.getTableName(), nodeID)

	if err != nil {
		return err
	}

	return ns.DB.Put(ns.getTableName(), nodeID, nodeData)
}

func (ns *Nodes) DeleteNode(nodeID []byte) error {
	return ns.DB.Delete(ns.getTableName(), nodeID)
}
