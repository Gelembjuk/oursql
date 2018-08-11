package consensus

import (
	"bytes"
	"crypto/ecdsa"
	"errors"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/dbquery"
	"github.com/gelembjuk/oursql/node/structures"
	"github.com/gelembjuk/oursql/node/transactions"
)

/*
* This structure is used to decide if a query ccan be executed by given pubkey
 */

type queryManager struct {
	DB      database.DBManager
	Logger  *utils.LoggerMan
	pubKey  []byte
	privKey ecdsa.PrivateKey
}

func (q queryManager) getQueryParser() dbquery.QueryProcessorInterface {
	return dbquery.NewQueryProcessor(q.DB, q.Logger)
}

func (q queryManager) getTransactionsManager() transactions.TransactionsManagerInterface {
	return transactions.NewManager(q.DB, q.Logger)
}

// New query from command line tool
// The method decides what to do next
// possible states:
// 0 and error . Query is wrong or execution failed or error when try to make TX if internal pub key used
// 1. Query doesn't need signature . It was executed .
// 2. Query needs signature and all other info. Data to sign is not yet preared (pubkey was not provided)
// 3. Query needs signature. TX was prepared and data to sign is retuned
// 4. Query needs signature. TX was created with provied signature
// 5. Query needs signature. TX was created with internal keys and completed
// PubKey is optional. It can be used to make the flow to be faster. In case if
// query needs external signature, this pubKey is used to build new TX and return data to sign by this key
// can return prepared TX with data to sign or complete TX. if TX is complete, it is added to the pool and query executed
// @return
// status int, txBytes []byte, datatosign []byte, transaction structure ref, error
func (q queryManager) NewQuery(sql string, pubKey []byte) (uint, []byte, []byte, *structures.Transaction, error) {
	return q.processQuery(sql, pubKey, true)
}

// Complete query execution. Accepts TX prepared with a request NewQuery and signed data
// private key must be corresponding to pub key used in NewQuery.
// SQL query in inside prepared TX. after it is verified, query can be finally executed
func (q queryManager) NewQuerySigned(txEncoded []byte, signature []byte) (*structures.Transaction, error) {
	return q.processQueryWithSignature(txEncoded, signature, true)
}

// execute new query and create transaction if needed . This provided private key to sign transaction if needed
// return complete TX. it is added to the pool and query executed. if TX is nil, it means query was executed without TX
func (q queryManager) NewQueryByNode(sql string, pubKey []byte, privKey ecdsa.PrivateKey) (uint, *structures.Transaction, error) {
	localError := func(err error) (uint, *structures.Transaction, error) {
		q.Logger.Trace.Printf("Return error: %s", err.Error())
		return SQLProcessingResultError, nil, err
	}

	q.Logger.Trace.Printf("Execute new SQL: %s", sql)

	r, txdata, datatosign, tx, err := q.processQuery(sql, pubKey, true)

	if err != nil {
		return localError(err)
	}
	q.Logger.Trace.Printf("Preparation done with status %d", r)
	if r == SQLProcessingResultExecuted ||
		r == SQLProcessingResultTranactionComplete ||
		r == SQLProcessingResultTranactionCompleteInternally {
		// no anymore actions are needed. Query was passed to mysql server
		// if transaction was created, it is already in a pool , if no than it is nil
		return r, tx, nil
	}
	if r == SQLProcessingResultPubKeyRequired {
		return localError(errors.New("PubKey is not provided or is wrong"))
	}
	// sign data and continue
	q.Logger.Trace.Printf("Sign new TX by %x", pubKey)
	signature, err := utils.SignDataByPubKey(pubKey, privKey, datatosign)

	if err != nil {
		return localError(err)
	}

	tx, err = q.NewQuerySigned(txdata, signature)

	if err != nil {
		return localError(err)
	}
	q.Logger.Trace.Printf("All fine. New TX: %x", tx.GetID())
	return SQLProcessingResultTranactionComplete, tx, nil
}

// DB proxy received new query .
// The query can contains comments with some additional instructions . this function should parse
// If error is returned, proxy will send the eror back to client.
// Error can contains special instructions related to data signing.
// returns transaction only in case if the object contains keys or client provided signature
// the TX should be added to pool by a proxy after success execution of the query
func (q queryManager) NewQueryFromProxy(sql string) (*structures.Transaction, error) {
	r, txdata, datatosign, tx, err := q.processQuery(sql, []byte{}, false)
	// formate error message
	if err != nil {
		return nil, err
	}
	if r == SQLProcessingResultExecuted ||
		r == SQLProcessingResultTranactionComplete ||
		r == SQLProcessingResultTranactionCompleteInternally {
		return tx, nil // no anymore actions are needed. Query can be passed to mysql server
	}
	// it is needed to return error of  specific formate. it an include TX and data to sign
	qp := q.getQueryParser()

	errStr, err := qp.FormatSpecialErrorMessage(r, txdata, datatosign)

	if err != nil {
		return nil, err
	}

	return nil, errors.New(errStr)
}
func (q queryManager) ExecuteOnBlockAdd(txlist []structures.Transaction) error {

	for _, tx := range txlist {
		if tx.IsSQLCommand() {
			q.Logger.Trace.Printf("Execute: %s", tx.GetSQLQuery())
			_, err := q.getQueryParser().ExecuteQuery(tx.GetSQLQuery())
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func (q queryManager) ExecuteOnBlockCancel(txlist []structures.Transaction) error {
	return nil
}

// ========================================================================================
// this does all work. It checks query, decides if ll data are present and creates transaction
// it can return prepared transaction and data to sign or return complete transaction if keys are set in the object
func (q queryManager) processQuery(sql string, pubKey []byte, executeifallowed bool) (uint, []byte, []byte, *structures.Transaction, error) {
	localError := func(err error) (uint, []byte, []byte, *structures.Transaction, error) {
		return SQLProcessingResultError, nil, nil, nil, err
	}
	qp := q.getQueryParser()
	// this will get sql type and data from comments. data can be pubkey, txBytes, signature
	qparsed, err := qp.ParseQuery(sql)

	if err != nil {
		return localError(err)
	}

	// maybe this query contains signature and txData from previous calls
	if len(qparsed.Signature) > 0 && len(qparsed.TransactionBytes) > 0 {
		// this is a case when signature and txdata were part of SQL comments.
		tx, err := q.processQueryWithSignature(qparsed.TransactionBytes, qparsed.Signature, executeifallowed)

		if err != nil {
			return localError(err)
		}

		return SQLProcessingResultTranactionComplete, nil, nil, tx, nil
	}

	needsTX, err := q.checkQueryNeedsTransaction(qparsed)

	if err != nil {
		return localError(err)
	}

	if !needsTX {
		if !executeifallowed {
			// no need to execute query. just return
			return SQLProcessingResultExecuted, nil, nil, nil, nil
		}
		// no need to have TX
		if qparsed.IsUpdate() {
			_, err := qp.ExecuteQuery(qparsed.SQL)
			if err != nil {
				return localError(err)
			}
		}
		return SQLProcessingResultExecuted, nil, nil, nil, nil
	}
	// decide which pubkey to use.

	// first priority for a key posted as argument, next is the key in SQL comment (parsed) and final is the key
	// provided to thi module
	if len(pubKey) == 0 {
		if len(qparsed.PubKey) > 0 {
			pubKey = qparsed.PubKey
		} else if len(q.pubKey) > 0 {
			pubKey = q.pubKey
		} else {
			// no pubkey to use. return notice about pubkey required
			return SQLProcessingResultPubKeyRequired, nil, nil, nil, nil
		}
	}

	// check if the key has permissions to execute this query
	hasPerm, err := q.checkExecutePermissions(qparsed, pubKey)

	if err != nil {
		return localError(err)
	}

	if !hasPerm {
		return localError(errors.New("No permissions to execute this query"))
	}

	amount, err := q.checkQueryNeedsPayment(qparsed)

	if err != nil {
		return localError(err)
	}
	// prepare SQL part of a TX
	sqlUpdate, err := qp.MakeSQLUpdateStructure(qparsed)

	if err != nil {
		return localError(err)
	}

	// find TX where thi refID was last updated and add it to sqlUpdate too
	// TODO

	var tx *structures.Transaction
	var datatosign []byte

	if amount > 0 {
		// prepare curency TX and add SQL part
		var txData []byte
		txData, datatosign, err = q.getTransactionsManager().PrepareNewSQLTransaction(pubKey, sqlUpdate, amount, "MINTER")
		if err != nil {
			return localError(err)
		}

		tx, err = structures.DeserializeTransaction(txData)
		if err != nil {
			return localError(err)
		}
	} else {
		// currrency part will be empty in new TX
		tx, err = structures.NewSQLTransaction(sqlUpdate, nil, nil)

		if err != nil {
			return localError(err)
		}
		datatosign, err = tx.PrepareSignData(pubKey, nil)

		if err != nil {
			return localError(err)
		}
	}
	txBytes, err := structures.SerializeTransaction(tx)
	if err != nil {
		return localError(err)
	}

	if len(q.pubKey) > 0 && bytes.Compare(q.pubKey, pubKey) == 0 {
		// transaction was created by internal pubkey. we have private key for it
		signature, err := utils.SignDataByPubKey(q.pubKey, q.privKey, datatosign)
		if err != nil {
			return localError(err)
		}

		tx, err = q.processQueryWithSignature(txBytes, signature, executeifallowed)

		if err != nil {
			return localError(err)
		}

		return SQLProcessingResultTranactionCompleteInternally, nil, nil, tx, nil
	}
	return SQLProcessingResultSignatureRequired, txBytes, datatosign, nil, nil
}

// check if this pubkey can execute this query
func (q queryManager) processQueryWithSignature(txEncoded []byte, signature []byte, executeifallowed bool) (*structures.Transaction, error) {
	tx, err := structures.DeserializeTransaction(txEncoded)

	if err != nil {
		return nil, err
	}
	q.Logger.Trace.Printf("Complete SQL TX")
	err = tx.CompleteTransaction(signature)

	if err != nil {
		return nil, err
	}
	q.Logger.Trace.Printf("Completed with ID: %x", tx.GetID())
	// verify
	// TODO

	q.Logger.Trace.Printf("Adding TX to pool")
	//return nil, errors.New("Temp err ")
	// add to pool
	// if fails , execute rollback ???
	// query wil be executed inside transactions manager before adding to a pool
	err = q.getTransactionsManager().ReceivedNewTransaction(tx, executeifallowed)

	if err != nil {
		return nil, err
	}
	return tx, nil
}

// check if this pubkey can execute this query
func (q queryManager) checkExecutePermissions(qp dbquery.QueryParsed, pubKey []byte) (bool, error) {
	return true, nil
}

// check if this query requires payment for execution. return number
func (q queryManager) checkQueryNeedsPayment(qp dbquery.QueryParsed) (float64, error) {
	return 0, nil
}

// check if this query must be added to transaction. all SELECT queries must be ignored.
// and some update queries can be ignored too. such queries are just executed
func (q queryManager) checkQueryNeedsTransaction(qp dbquery.QueryParsed) (bool, error) {

	if qp.IsSelect() {
		return false, nil
	}
	// transaction for any update
	return true, nil
}
