package consensus

import (
	"bytes"
	"crypto/ecdsa"
	"errors"

	"github.com/gelembjuk/oursql/lib"
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
	config  *ConsensusConfig
}

type processQueryResponse struct {
	status       uint
	txdata       []byte
	stringtosign []byte
	tx           *structures.Transaction
}

func (q queryManager) getQueryParser() dbquery.QueryProcessorInterface {
	return dbquery.NewQueryProcessor(q.DB, q.Logger)
}

func (q queryManager) getTransactionsManager() transactions.TransactionsManagerInterface {
	return transactions.NewManager(q.DB, q.Logger, q.config.GetInfoForTransactions())
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
	result, err := q.processQuery(sql, pubKey, transactions.TXFlagsExecute)
	return result.status, result.txdata, result.stringtosign, result.tx, err
}

// Complete query execution. Accepts TX prepared with a request NewQuery and signed data
// private key must be corresponding to pub key used in NewQuery.
// SQL query in inside prepared TX. after it is verified, query can be finally executed
func (q queryManager) NewQuerySigned(txEncoded []byte, signature []byte) (*structures.Transaction, error) {
	return q.processQueryWithSignature(txEncoded, signature, transactions.TXFlagsExecute)
}

// execute new query and create transaction if needed . This provided private key to sign transaction if needed
// return complete TX. it is added to the pool and query executed. if TX is nil, it means query was executed without TX
func (q queryManager) NewQueryByNode(sql string, pubKey []byte, privKey ecdsa.PrivateKey) (uint, *structures.Transaction, error) {
	localError := func(err error) (uint, *structures.Transaction, error) {
		q.Logger.Trace.Printf("Return error: %s", err.Error())
		return SQLProcessingResultError, nil, err
	}

	q.Logger.Trace.Printf("Execute new SQL: %s", sql)

	result, err := q.processQuery(sql, pubKey, transactions.TXFlagsExecute)

	if err != nil {
		return localError(err)
	}
	q.Logger.Trace.Printf("Preparation done with status %d", result.status)
	if result.status == SQLProcessingResultExecuted ||
		result.status == SQLProcessingResultTranactionComplete ||
		result.status == SQLProcessingResultTranactionCompleteInternally {
		// no anymore actions are needed. Query was passed to mysql server
		// if transaction was created, it is already in a pool , if no than it is nil
		return result.status, result.tx, nil
	}
	if result.status == SQLProcessingResultPubKeyRequired {
		return localError(errors.New("PubKey is not provided or is wrong"))
	}
	// sign data and continue
	q.Logger.Trace.Printf("Sign new TX by %x", pubKey)
	signature, err := utils.SignDataByPubKey(pubKey, privKey, result.stringtosign)

	if err != nil {
		return localError(err)
	}

	tx, err := q.NewQuerySigned(result.txdata, signature)

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
// TODO replace error and code with custom errror structure containing a code
func (q queryManager) NewQueryFromProxy(sql string) (result QueryFromProxyResult) {
	result.Status = 0 // error

	qpresult, err := q.processQuery(sql, []byte{}, transactions.TXFlagsNoPoool /*don't add to a pool*/)
	// formate error message
	if err != nil {
		result.ErrorCode = 4
		result.Error = err
		return
	}
	if qpresult.status == SQLProcessingResultExecuted ||
		qpresult.status == SQLProcessingResultTranactionComplete ||
		qpresult.status == SQLProcessingResultTranactionCompleteInternally ||
		qpresult.status == SQLProcessingResultCanBeExecuted {

		result.Status = 1 // final

		if qpresult.status == SQLProcessingResultCanBeExecuted {
			result.Status = 3 // pass query to server
		} else {
			result.ReplaceQuery = qpresult.tx.GetSQLQuery()
		}

		result.TX = qpresult.tx
		return // no anymore actions are needed. Query can be passed to mysql server
	}
	// it is needed to return error of  specific formate. it an include TX and data to sign

	if qpresult.status == SQLProcessingResultPubKeyRequired {
		result.ErrorCode = 2
		result.Error = errors.New("Error(2): Public Key required")
		return
	}

	if qpresult.status == SQLProcessingResultSignatureRequired {
		/*
			result.ErrorCode = 3
			result.Error = errors.New("Error(3): Signature required")
			return
		*/
		result.TXData = qpresult.txdata
		result.StringToSign = qpresult.stringtosign
		result.Status = 2 // signature data prepared

		return
	}
	result.ErrorCode = 5
	result.Error = errors.New("Unknown query status")

	return
}

// this is executed to add a list of transactions back to unapproved list (pool)
// it is used to add transactions back to pool from canceled blocks in case if branches are switched
// some SQL transactions can not be added back because base TX was used by other tx that is in a block now
// this function will try to update that TX , but only in a case if original TX was signed by this node
func (q queryManager) RepeatTransactionsFromCanceledBlocks(txList []structures.Transaction) error {
	for _, tx := range txList {
		q.Logger.Trace.Printf("Try to add back TX %x", tx.GetID())

		if tx.IsSQLCommand() {
			q.Logger.Trace.Printf("SQL %s : %s", string(tx.SQLCommand.Query), string(tx.SQLCommand.ReferenceID))
			q.Logger.Trace.Printf("TX based on %x", tx.SQLBaseTX)
		} else {
			q.Logger.Trace.Printf("It is currency TX")
		}

		err := q.getTransactionsManager().ReceivedNewTransaction(&tx, transactions.TXFlagsExecute)

		if err != nil {
			q.Logger.Trace.Printf("Erro adding TX back %x %s", tx.GetID(), err.Error())
			// check if error is "base tx already used" and try to resign this TX
			if verr, ok := err.(*transactions.TXVerifyError); ok {
				q.Logger.Trace.Printf("Error verify of kinf %s", verr.GetKind())

				if verr.IsKind(transactions.TXSQLBaseDifferentError) {
					q.Logger.Trace.Printf("Is base diff")
					err := q.tryToRepeatTransactionResigned(&tx, verr.TX)

					if err != nil {
						q.Logger.Trace.Printf("Error on Repeat adding TX back %x %s", tx.GetID(), err.Error())
					} else {
						q.Logger.Trace.Printf("TX added back after repeat %x", tx.GetID())
					}
				} else {
					q.Logger.Trace.Printf("Is NOT base diff")
				}
			} else {
				q.Logger.Trace.Printf("Is NOT verify error")
			}
		} else {
			q.Logger.Trace.Printf("TX added back %x", tx.GetID())
		}
	}
	return nil
}

// ========================================================================================
// this does all work. It checks query, decides if ll data are present and creates transaction
// it can return prepared transaction and data to sign or return complete transaction if keys are set in the object
func (q queryManager) processQuery(sql string, pubKey []byte, flags int) (result processQueryResponse, err error) {
	q.Logger.Trace.Println("processQuery " + sql)
	qp := q.getQueryParser()
	// this will get sql type and data from comments. data can be pubkey, txBytes, signature
	qparsed, err := qp.ParseQuery(sql)

	if err != nil {
		return
	}

	// maybe this query contains signature and txData from previous calls
	if len(qparsed.Signature) > 0 && len(qparsed.TransactionBytes) > 0 {
		// this is a case when signature and txdata were part of SQL comments.
		result.tx, err = q.processQueryWithSignature(qparsed.TransactionBytes, qparsed.Signature, flags)

		if err != nil {
			return
		}
		result.status = SQLProcessingResultTranactionComplete

		return
	}

	needsTX, err := q.checkQueryNeedsTransaction(qparsed)

	if err != nil {
		return
	}

	if !needsTX {
		if flags&transactions.TXFlagsExecute == 0 {
			// no need to execute query. just return
			result.status = SQLProcessingResultCanBeExecuted
			return
		}

		// no need to have TX
		if qparsed.IsUpdate() {

			_, err = qp.ExecuteQuery(qparsed.SQL)
			if err != nil {
				return
			}
		}
		result.status = SQLProcessingResultExecuted

		return
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
			result.status = SQLProcessingResultPubKeyRequired
			return
		}
	}

	// check if the key has permissions to execute this query
	hasPerm, err := q.checkExecutePermissions(qparsed, pubKey)

	if err != nil {
		return
	}

	if !hasPerm {
		err = errors.New("No permissions to execute this query")
		return
	}

	amount, err := q.checkQueryNeedsPayment(qparsed)

	if err != nil {
		return
	}
	// prepare SQL part of a TX
	// this builds RefID for a TX update
	sqlUpdate, err := qp.MakeSQLUpdateStructure(qparsed)

	if err != nil {
		return
	}

	// prepare curency TX and add SQL part

	result.txdata, result.stringtosign, err = q.getTransactionsManager().PrepareNewSQLTransaction(pubKey, sqlUpdate, amount, "MINTER")

	if err != nil {
		return
	}

	result.tx, err = structures.DeserializeTransaction(result.txdata)

	if err != nil {
		return
	}

	if len(q.pubKey) > 0 && bytes.Compare(q.pubKey, pubKey) == 0 {
		// transaction was created by internal pubkey. we have private key for it
		var signature []byte
		signature, err = utils.SignDataByPubKey(q.pubKey, q.privKey, result.stringtosign)
		if err != nil {
			return
		}

		result.tx, err = q.processQueryWithSignature(result.txdata, signature, flags)

		if err != nil {
			return
		}
		result.status = SQLProcessingResultTranactionCompleteInternally
		return
	}
	result.status = SQLProcessingResultSignatureRequired
	return
}

// check if this pubkey can execute this query
func (q queryManager) processQueryWithSignature(txEncoded []byte, signature []byte, flags int) (*structures.Transaction, error) {
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

	// add to pool
	// query wil be executed inside transactions manager before adding to a pool
	err = q.getTransactionsManager().ReceivedNewTransaction(tx, flags)

	if err != nil {
		return nil, err
	}
	return tx, nil
}

// check if this pubkey can execute this query
func (q queryManager) checkExecutePermissions(qp dbquery.QueryParsed, pubKey []byte) (bool, error) {
	// check sonsensus rules
	if !qp.IsUpdate() {
		return true, nil
	}

	hasCustom, allow, err := q.checkExecutePermissionsAsTable(qp, pubKey)

	if err != nil {
		return false, err
	}

	if hasCustom {
		return allow, nil
	}

	if qp.Structure.GetKind() == lib.QueryKindCreate {
		if !q.config.AllowTableCreate {
			return false, nil
		}
	}

	if qp.Structure.GetKind() == lib.QueryKindDrop {
		if !q.config.AllowTableDrop {
			return false, nil
		}
	}

	if qp.Structure.GetKind() == lib.QueryKindDelete {
		if !q.config.AllowRowDelete {
			return false, nil
		}
	}

	return true, nil
}

// check custom rule for the table about permissions
func (q queryManager) checkExecutePermissionsAsTable(qp dbquery.QueryParsed, pubKey []byte) (hasCustom bool, allow bool, err error) {
	hasCustom = false
	// check sonsensus rules
	if !qp.IsUpdate() {
		return
	}

	if q.config.TableRules == nil {
		// no any rules
		return
	}

	for _, t := range q.config.TableRules {
		if t.Table != qp.Structure.GetTable() {
			continue
		}
		if !t.AllowRowDelete && qp.Structure.GetKind() == lib.QueryKindDelete {
			hasCustom = true
			allow = false
			return
		}

		if !t.AllowRowInsert && qp.Structure.GetKind() == lib.QueryKindInsert {
			hasCustom = true
			allow = false
			return
		}

		if !t.AllowRowUpdate && qp.Structure.GetKind() == lib.QueryKindUpdate {
			hasCustom = true
			allow = false
			return
		}

		if !t.AllowTableCreate && qp.Structure.GetKind() == lib.QueryKindCreate {
			hasCustom = true
			allow = false
			return
		}
		// has custom rule and operaion is not disabled
		hasCustom = true
		allow = true
		return
	}

	return
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

	if qp.IsUpdateOther() {
		// updates that can not be supported
		return false, nil
	}

	for _, t := range q.config.UnmanagedTables {
		if qp.Structure.GetTable() == t {
			// no any transactions for this table
			return false, nil
		}
	}

	// transaction for any update
	return true, nil
}

// check if this query must be added to transaction. all SELECT queries must be ignored.
// and some update queries can be ignored too. such queries are just executed
func (q queryManager) tryToRepeatTransactionResigned(tx *structures.Transaction, newSQLBaseTX []byte) error {
	if !tx.IsSQLCommand() {
		return errors.New("Can repeat only SQL transactions")
	}
	// check if there is priate key in consensus module and it TX pubkey is same
	if len(q.pubKey) > 0 && bytes.Compare(q.pubKey, tx.ByPubKey) == 0 {
		q.Logger.Trace.Printf("Signed by same pubkey as this node has %x", q.pubKey)
		// change base TX and rebuild the tx
		tx.SetSQLPreviousTX(newSQLBaseTX)

		txdata, stringtosign, err := q.getTransactionsManager().PrepareSQLTransactionSignatureData(tx)

		if err != nil {
			return err
		}

		signature, err := utils.SignDataByPubKey(q.pubKey, q.privKey, stringtosign)

		if err != nil {
			return err
		}

		tx, err = q.processQueryWithSignature(txdata, signature, transactions.TXFlagsExecute /*execute query if all is fine*/)

		if err != nil {
			return err
		}
		return nil
	}
	if len(q.pubKey) == 0 {
		return errors.New("The node has no private key to sign transactions")
	}
	return errors.New("Transaction was signed by different public key from current node key")
}
