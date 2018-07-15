package dbquery

type QueryParsed struct {
	SQL              string
	SQLKind          string
	ReferenceID      string
	PubKey           []byte
	Signature        []byte
	TransactionBytes []byte
	Structure        QueryStructure
}

type QueryStructure struct {
	Kind        string
	Table       string
	ReferenceID string
}

func (qp QueryParsed) IsSelect() bool {
	return qp.SQLKind == QueryKindSelect
}

func (qp QueryParsed) IsUpdate() bool {
	return qp.SQLKind != QueryKindSelect
}

/*

type mySQLQueryManager struct {
	DB     database.DBManager
	Logger *utils.LoggerMan
}
// make new trsnaction, signs it and adds to a pool. if private key is stored on a server
func (dbq *mySQLQueryManager) NewSQLQueryTransaction(PubKey []byte, privKey ecdsa.PrivateKey, sqlcommand string) (*structures.Transaction, error) {
	newTXData, signData, err := dbq.StartSQLQueryTransaction(PubKey, sqlcommand)

	if err != nil {
		return nil, err
	}

	if newTXData == nil {
		// query was executed without a TX
		return nil, nil
	}

	signature, err := utils.SignDataByPubKey(PubKey, privKey, signData)

	if err != nil {
		return nil, err
	}

	return dbq.CompleteSQLQueryTransaction(newTXData, signature)
}

// makes new transaction. builds data to sign
// we need to check
// - if a query is correct.
// - if this pubKey can execute this query
// - if query needs payment for execution if there are anough money
// when all passed, create new TX. query is not yet executed. it will be on complete when TX signed
func (dbq *mySQLQueryManager) StartSQLQueryTransaction(PubKey []byte, sqlcommand string) ([]byte, []byte, error) {
	localError := func(err error) ([]byte, []byte, error) {
		return nil, nil, err
	}
	qp, err := dbq.getQueryProcessor()

	if err != nil {
		return localError(err)
	}

	needtx, err := qp.CheckQueryNeedsTransaction(sqlcommand)

	if err != nil {
		return localError(err)
	}

	if !needtx {
		// execuute this query
		err := qp.ExecteQuery(sqlcommand)

		if err != nil {
			return localError(err)
		}
		return nil, nil, nil
	}

	nsql, err := qp.CheckQueryCanBeExecuted(sqlcommand)

	if err != nil {
		return localError(err)
	}

	if nsql != "" {
		sqlcommand = nsql
	}

	canexecute, err := qp.CheckExecutePermissions(sqlcommand, PubKey)

	if err != nil {
		return localError(err)
	}

	if !canexecute {
		return localError(errors.New("No permissions to execute"))
	}

	amount, err := qp.CheckQueryNeedsPayment(sqlcommand)

	if err != nil {
		return localError(err)
	}

	return nil, nil, nil
}

// completes transaction. add signature, verify.
// if all fine, execute the query . if success return new transaction
func (dbq *mySQLQueryManager) CompleteSQLQueryTransaction(txData []byte, signatire []byte) (*structures.Transaction, error) {
	return nil, nil
}
*/
