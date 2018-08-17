package database

const dataReferencesTable = "rowstotransactions"

type dataReferences struct {
	DB                  *MySQLDB
	dataReferencesTable string
}

func (dr *dataReferences) getDataReferencesTable() string {
	if dr.dataReferencesTable == "" {
		dr.dataReferencesTable = dr.DB.tablesPrefix + dataReferencesTable
	}
	return dr.dataReferencesTable
}

// Init database
func (dr *dataReferences) InitDB() error {
	return dr.DB.CreateTable(dr.getDataReferencesTable(), "VARBINARY(100)", "VARBINARY(100)")
}

// transacet tables
func (dr *dataReferences) TruncateDB() error {
	return dr.DB.Truncate(dr.getDataReferencesTable())
}

// Save link between TX and block hash
func (dr *dataReferences) SetTXForRefID(RefID []byte, txID []byte) error {
	return dr.DB.Put(dr.getDataReferencesTable(), RefID, txID)
}

// Get block hash for TX
func (dr *dataReferences) GetTXForRefID(RefID []byte) ([]byte, error) {
	return dr.DB.Get(dr.getDataReferencesTable(), RefID)
}

// Delete link between TX and a block hash
func (dr *dataReferences) DeleteRefID(RefID []byte) error {
	return dr.DB.Delete(dr.getDataReferencesTable(), RefID)
}
