package transactions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/dbquery"
	"github.com/gelembjuk/oursql/node/structures"
)

const maxCountOfTransactionInMemoryCache = 20000
const transactionsCacheEnable = true

var transactionsCache map[string]structures.Transaction
var transactionsCacheLock *sync.Mutex

type unApprovedTransactions struct {
	DB     database.DBManager
	Logger *utils.LoggerMan
}

type foreachFunction func(tx *structures.Transaction) (bool, error)

// Lock Transactions Cache access
func (u unApprovedTransactions) lockCache() {
	if transactionsCacheLock == nil {
		transactionsCacheLock = &sync.Mutex{}
	}
	transactionsCacheLock.Lock()
}

// Unlock transactions cache access
func (u unApprovedTransactions) unlockCache() {
	if transactionsCacheLock == nil {
		// it was not locked
		return
	}
	transactionsCacheLock.Unlock()
}

func (u unApprovedTransactions) renewCacheIfNeeded() error {
	if !transactionsCacheEnable {
		return nil
	}

	if transactionsCache != nil {
		return nil
	}

	return u.renewCache()
}

// Loads all Txs from DB to memory
func (u *unApprovedTransactions) renewCache() error {
	if !transactionsCacheEnable {
		return nil
	}
	u.Logger.Trace.Println("Renwe TX cache")
	u.lockCache()
	defer u.unlockCache()
	// get count of TX in pool
	transactionsCache = nil

	c, err := u.GetCount()

	if err != nil {
		return err
	}

	if c > maxCountOfTransactionInMemoryCache {
		return newTXPoolCacheNoMemoryError("Too many transactions to use pool cache")
	}
	utdb, err := u.DB.GetUnapprovedTransactionsObject()

	if err != nil {
		return err
	}
	transactionsCache = make(map[string]structures.Transaction, 0)

	allPairs, err := utdb.GetAll()

	if err != nil {
		u.Logger.Trace.Println("Loading error %s ", err.Error())
		return err
	}

	for _, row := range allPairs {
		// TX is serialised in avalue
		tx, err := structures.DeserializeTransaction(row[1])

		if err != nil {
			return err
		}
		//u.Logger.Trace.Printf("TX adding to cache %x", tx.GetID())
		transactionsCache[tx.GetIDString()] = *tx
	}

	return nil
}

func (u unApprovedTransactions) forEachTransaction(callback foreachFunction) error {
	// it i needed to go over all tranactions in cache and check each of them

	if transactionsCacheEnable {

		u.lockCache()
	}

	if transactionsCacheEnable && transactionsCache != nil {
		defer u.unlockCache()

		for _, txC := range transactionsCache {
			tx := txC
			stop, err := callback(&tx)

			if err != nil {
				return err
			}

			if stop {
				break
			}
		}
	} else {
		if transactionsCacheEnable {
			u.unlockCache()
		}

		utdb, err := u.DB.GetUnapprovedTransactionsObject()

		if err != nil {
			return err
		}
		err = utdb.ForEach(func(txid, txBytes []byte) error {
			tx, err := structures.DeserializeTransaction(txBytes)

			if err != nil {
				return err
			}

			_, err = callback(tx)

			return err
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// Check if transaction inputs are pointed to some prepared transactions.
// Check conflicts too. Same output can not be repeated twice
func (u *unApprovedTransactions) CheckInputsArePrepared(inputs map[int]structures.TXCurrencyInput, inputTXs map[int]*structures.Transaction) error {
	checked := map[string][]int{}

	for vinInd, vin := range inputs {
		// look if not yet checked

		txstr := hex.EncodeToString(vin.Txid)

		if outs, ok := checked[txstr]; ok {
			// tx was checked
			for _, out := range outs {
				if out == vin.Vout {
					// this output was already used in outher input
					return errors.New(fmt.Sprintf("Duplicate usage of transaction output: %s - %d", txstr, out))
				}
			}
		}

		// check if this transaction exists
		tx, err := u.GetIfExists(vin.Txid)

		if err != nil {
			return err
		}

		if tx == nil {
			return NewTXVerifyError("Input transaction is not found in prepared to approve", TXVerifyErrorNoInput, vin.Txid)
		}
		//u.Logger.Trace.Printf("FOund TX %x %x for ind %d", vin.Txid, tx.GetID(), vinInd)
		inputTXs[vinInd] = tx
		checked[txstr] = append(checked[txstr], vin.Vout)
	}
	return nil
}

// Check if transaction inputs are pointed to some non approved transactions.
// That are listed in a block before this transactions
// Receives list of inputs and previous transactions
// and input transactions for this tx
// Check conflicts too. Same output can not be repeated twice

func (u *unApprovedTransactions) CheckCurrencyInputsWereBefore(
	inputs map[int]structures.TXCurrencyInput,
	prevTXs []structures.Transaction,
	inputTXs map[int]*structures.Transaction) (map[int]*structures.Transaction, error) {

	//u.Logger.Trace.Printf("Check inputs were before. inputs %d, prev %d", len(inputs), len(prevTXs))

	checked := map[string][]int{}

	for vind, vin := range inputs {
		// look if not yet checked

		txstr := hex.EncodeToString(vin.Txid)

		if outs, ok := checked[txstr]; ok {
			// tx was checked
			for _, out := range outs {
				if out == vin.Vout {
					// this output was already used in outher input
					return inputTXs, errors.New("Duplicate usage of transaction output")
				}
			}
		}

		// check if this transaction exists in the list
		exists := false

		for _, tx := range prevTXs {
			u.Logger.Trace.Printf("compare input %x and previous %x", vin.Txid, tx.GetID())
			if bytes.Compare(vin.Txid, tx.GetID()) == 0 {
				inputTXs[vind] = &tx
				exists = true
				break
			}
		}

		if !exists {
			return inputTXs, NewTXVerifyError("Input transaction is not found in prepared to approve", TXVerifyErrorNoInput, vin.Txid)
		}

		checked[txstr] = append(checked[txstr], vin.Vout)
	}
	return inputTXs, nil
}

// Returns pending transations info prepared by address
// Return contains:
// List of all inputs used by this PubKeyHash in his pending TXs
// List of Outputs that were not yet used in any input (returned in the first list).
//		this are outputs that can be still used in new TX
// List of inputs based on approved outputs (sub list of the first list). From the first list
//		we dropped inputs where otput is from pending TX
func (u *unApprovedTransactions) GetCurrencyTXsPreparedBy(PubKeyHash []byte) ([]structures.TXCurrencyInput,
	[]*structures.TXOutputIndependent, []structures.TXCurrencyInput, error) {

	inputs := []structures.TXCurrencyInput{}
	outputs := []*structures.TXOutputIndependent{}

	//u.Logger.Trace.Println("GetCurrencyTXsPreparedBy")
	// goes over all pending (unconfirmed) transactions in the cache
	// check every input for every TX and adds to "inputs" if that input was signed by this pub key

	err := u.forEachTransaction(func(tx *structures.Transaction) (bool, error) {

		sender := []byte{}

		if !tx.IsCoinbaseTransfer() {
			sender = tx.ByPubKey

			if tx.CreatedByPubKeyHash(PubKeyHash) {
				inputs = append(inputs, tx.Vin...)
			}

		}
		for indV, vout := range tx.Vout {
			if vout.IsLockedWithKey(PubKeyHash) {
				voutind := structures.TXOutputIndependent{}
				// we are settings serialised transaction in place of block hash
				// we don't have a block for such transaction , but we need full transaction later
				txBytes, err := structures.SerializeTransaction(tx)

				if err != nil {
					return false, err
				}

				voutind.LoadFromSimple(vout, tx.ID, indV, sender, tx.IsCoinbaseTransfer(), txBytes)

				// "outputs" contains list of outputs of transations in the pending cache
				// we need it to know later which outputs and used as inputs for other pending transactions
				// and to know their out value
				outputs = append(outputs, &voutind)
			}
		}
		return false, nil
	})

	if err != nil {
		return nil, nil, nil, err
	}

	// outputs not yet used in other pending transactions
	// not yet spent outputs of pending transactions
	realoutputs := []*structures.TXOutputIndependent{}

	// inputs based on approved transactions. sublist of "inputs"
	approvedinputs := []structures.TXCurrencyInput{}

	for _, vout := range outputs {
		used := false
		for _, vin := range inputs {
			if bytes.Compare(vin.Txid, vout.TXID) == 0 && vin.Vout == vout.OIndex {
				// this output is already used in other pending transaction
				used = true
				break
			}
		}
		if !used {
			// add to thi list only if output was not used as input in any pending TX
			realoutputs = append(realoutputs, vout)
		}
	}
	// find inputs from TXs outs that were already approved
	for _, vin := range inputs {
		pendingout := false

		for _, vout := range outputs {
			if bytes.Compare(vin.Txid, vout.TXID) == 0 && vin.Vout == vout.OIndex {
				// this input uses pending output
				pendingout = true
				break
			}
		}

		if !pendingout {
			// this input is not output of any pending TX. so, we presume it is output of
			// approved TX
			approvedinputs = append(approvedinputs, vin)
		}
	}
	return inputs, realoutputs, approvedinputs, nil
}

// Get input value for TX in the cache
func (u *unApprovedTransactions) GetInputValue(input structures.TXCurrencyInput) (float64, error) {
	//u.Logger.Trace.Printf("Find TX %x in unapproved", input.Txid)
	tx, err := u.GetIfExists(input.Txid)

	if err != nil {
		return 0, err
	}

	if tx == nil {
		return 0, errors.New("TX not found in cache of unapproved")
	}

	return tx.Vout[input.Vout].Value, nil
}

// Check if transaction exists in a cache of unapproved
func (u *unApprovedTransactions) GetIfExists(txid []byte) (*structures.Transaction, error) {
	u.lockCache()

	if transactionsCacheEnable && transactionsCache != nil {

		defer u.unlockCache()

		txIDString := fmt.Sprintf("%x", txid)

		if tx, ok := transactionsCache[txIDString]; ok {
			//u.Logger.Trace.Printf("Found TX in cache %x", tx.GetID())
			return &tx, nil
		}

		return nil, nil
	}

	u.unlockCache()

	utdb, err := u.DB.GetUnapprovedTransactionsObject()

	if err != nil {
		return nil, err
	}

	txBytes, err := utdb.GetTransaction(txid)

	if err != nil {
		return nil, err
	}

	if len(txBytes) == 0 {
		return nil, nil
	}

	tx, err := structures.DeserializeTransaction(txBytes)

	if err != nil {
		return nil, err
	}
	//u.Logger.Trace.Printf("Found TX %x in DB", tx.GetID())
	return tx, nil

}

// Get all unapproved transactions
func (u *unApprovedTransactions) GetTransactions(number int) ([]*structures.Transaction, error) {

	txset := []*structures.Transaction{}

	totalnumber := 0
	//u.Logger.Trace.Println("GetTransactions")

	err := u.forEachTransaction(func(tx *structures.Transaction) (bool, error) {

		txset = append(txset, tx)
		totalnumber++

		if totalnumber >= number {
			// time to exit the loop. we don't need more
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return nil, err
	}

	// we need to sort transactions. oldest should be first
	sort.Sort(structures.Transactions(txset))
	return txset, nil
}

// Get all unapproved transactions filtered by list of Txs to skip
func (u *unApprovedTransactions) GetTransactionsFilteredByList(number int, ignoreTransactions [][]byte) ([]*structures.Transaction, error) {

	txset := []*structures.Transaction{}

	totalnumber := 0
	//u.Logger.Trace.Println("GetTransactionsFilteredByList")

	err := u.forEachTransaction(func(tx *structures.Transaction) (bool, error) {
		for _, txF := range ignoreTransactions {
			if bytes.Compare(txF, tx.GetID()) == 0 {
				return false, nil
			}
		}

		txset = append(txset, tx)
		totalnumber++

		if totalnumber >= number {
			// time to exit the loop. we don't need more
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return nil, err
	}

	// we need to sort transactions. oldest should be first
	sort.Sort(structures.Transactions(txset))
	return txset, nil
}

// Get all unapproved transactions filtered by create time and list to skip. Return only more recent
func (u *unApprovedTransactions) GetTransactionsFilteredByTime(number int,
	minCreateTime int64, ignoreTransactions [][]byte) ([]*structures.Transaction, error) {

	txset := []*structures.Transaction{}

	totalnumber := 0
	u.Logger.Trace.Println("GetTransactionsFilteredByTime")

	err := u.forEachTransaction(func(tx *structures.Transaction) (bool, error) {
		for _, txF := range ignoreTransactions {
			if bytes.Compare(txF, tx.GetID()) == 0 {
				return false, nil
			}
		}

		if tx.GetTime() < minCreateTime {
			return false, nil
		}

		txset = append(txset, tx)
		totalnumber++

		if totalnumber >= number {
			// time to exit the loop. we don't need more
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return nil, err
	}

	// we need to sort transactions. oldest should be first
	sort.Sort(structures.Transactions(txset))
	return txset, nil
}

// Get number of unapproved transactions in a cache
func (u *unApprovedTransactions) GetCount() (int, error) {
	utdb, err := u.DB.GetUnapprovedTransactionsObject()

	if err != nil {
		return 0, err
	}

	return utdb.GetCount()
}

// Get number of unapproved transactions in a cache, but ignoring given list of transactions
func (u *unApprovedTransactions) GetCountFiltered(ignoreTransactions [][]byte) (int, error) {

	count := 0
	//u.Logger.Trace.Println("GetCountFiltered")

	err := u.forEachTransaction(func(tx *structures.Transaction) (bool, error) {
		for _, txF := range ignoreTransactions {
			if bytes.Compare(txF, tx.GetID()) == 0 {
				return false, nil
			}
		}

		count = count + 1

		return false, nil
	})
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Add new transaction for the list of unapproved
// Before to call this function we checked that transaction is valid
// Now we need to check if there are no conflicts with other transactions in the cache
func (u *unApprovedTransactions) Add(txadd *structures.Transaction) error {
	u.Logger.Trace.Printf("Adding to pool in DB man %x", txadd.GetID())
	conflicts, err := u.DetectConflictsForNew(txadd)

	if err != nil {
		return err
	}

	if conflicts != nil {
		return errors.New(fmt.Sprintf("The transaction conflicts with other prepared transaction: %x", conflicts.GetID()))
	}

	utdb, err := u.DB.GetUnapprovedTransactionsObject()

	if err != nil {
		return err
	}

	u.Logger.Trace.Printf("adding TX to unappr %x", txadd.GetID())

	txser, err := structures.SerializeTransaction(txadd)

	if err != nil {
		u.Logger.Trace.Printf("err 1 %s", err.Error())
		return err
	}

	err = utdb.PutTransaction(txadd.GetID(), txser)

	if err != nil {
		u.Logger.Trace.Printf("err 2 %s", err.Error())
		return errors.New("Adding new transaction to unapproved cache: " + err.Error())
	}

	if transactionsCacheEnable && transactionsCache != nil {
		u.lockCache()
		defer u.unlockCache()

		transactionsCache[string(txadd.GetIDString())] = *txadd
		//u.Logger.Trace.Printf("Added TX to TX cache %x %s", txadd.GetID(), txadd.GetIDString())
	}

	return nil
}

/*
* Delete transaction from a cache. When transaction becomes part ofa block
 */
func (u *unApprovedTransactions) Delete(txid []byte) (bool, error) {
	//u.Logger.Trace.Printf("Remove TX from unapproved %x", txid)
	utdb, err := u.DB.GetUnapprovedTransactionsObject()

	if err != nil {
		u.Logger.Trace.Printf("Remove TX from unapproved error %s", err.Error())
		return false, err
	}

	txBytes, err := utdb.GetTransaction(txid)

	if err != nil {

		return false, err
	}

	if len(txBytes) > 0 {
		err = utdb.DeleteTransaction(txid)

		if err != nil {
			return false, err
		}
		if transactionsCacheEnable {
			// remove also from cache
			u.lockCache()
			defer u.unlockCache()

			txIDString := fmt.Sprintf("%x", txid)
			//u.Logger.Trace.Printf("Delete TX from cache %s", txIDString)
			if transactionsCache != nil {
				delete(transactionsCache, txIDString)
			}
		}

		return true, nil
	}

	return false, nil
}

/*
* Remove given blocks transavtions from unapproved . For case when list of blocks are added to primary blockchain branch
 */
func (u *unApprovedTransactions) DeleteFromBlocks(blocks []*structures.Block) error {
	for _, block := range blocks {

		err := u.DeleteFromBlock(block)

		if err != nil {

			return err
		}
	}
	return nil
}

/*
* Remove all transactions from this cache listed in a block.
* Is used when new block added and transactions are approved now
 */
func (u *unApprovedTransactions) DeleteFromBlock(block *structures.Block) error {
	// try to delete each transaction from this block
	//u.Logger.Trace.Printf("UnApprTXs: remove on block add %x", block.Hash)

	for _, tx := range block.Transactions {
		if !tx.IsCoinbaseTransfer() {
			u.Delete(tx.GetID())
		}
	}

	return nil
}

// Is used for cases when it is needed to do something with all cached transactions.
// For example, to print them.

func (u *unApprovedTransactions) forEachUnapprovedTransaction(callback UnApprovedTransactionCallbackInterface) (int, error) {
	utdb, err := u.DB.GetUnapprovedTransactionsObject()

	if err != nil {
		return 0, err
	}
	_, err = utdb.GetCount()

	total := 0

	err = u.forEachTransaction(func(tx *structures.Transaction) (bool, error) {
		callback(hex.EncodeToString(tx.GetID()), tx.String())
		total++

		return false, nil
	})
	if err != nil {
		return 0, err
	}

	return total, nil
}

// Check if this new transaction conflicts with any other transaction in the cache
// It is not allowed 2 prepared transactions have same inputs
// we return first found transaction taht conflicts
func (u *unApprovedTransactions) DetectConflictsForNew(txcheck *structures.Transaction) (*structures.Transaction, error) {
	// it i needed to go over all tranactions in cache and check each of them if input is same as in this tx

	//u.Logger.Trace.Println("DetectConflictsForNew")
	var txconflicts *structures.Transaction

	err := u.forEachTransaction(func(txexi *structures.Transaction) (bool, error) {

		if bytes.Compare(txexi.GetID(), txcheck.GetID()) == 0 {
			return false, nil
		}

		for _, vin := range txcheck.Vin {
			for _, vine := range txexi.Vin {
				if bytes.Compare(vin.Txid, vine.Txid) == 0 && vin.Vout == vine.Vout {
					// this is same input structures. it is conflict
					txconflicts = txexi

					return true, nil
				}
			}
		}
		if txcheck.IsSQLCommand() && txexi.IsSQLCommand() && bytes.Compare(txcheck.GetID(), txexi.GetID()) != 0 {
			// check if there is SQL conflict
			// SQL conflict can be if same base transaction and same ReferenceID
			if len(txexi.GetSQLBaseTX()) > 0 && len(txexi.SQLCommand.ReferenceID) > 0 &&
				bytes.Compare(txexi.GetSQLBaseTX(), txcheck.GetSQLBaseTX()) == 0 &&
				bytes.Compare(txexi.SQLCommand.ReferenceID, txcheck.SQLCommand.ReferenceID) == 0 {

				u.Logger.Trace.Printf("Same base TX and RefID for %x and %x", txcheck.GetID(), txexi.GetID())

				txconflicts = txexi
				return true, nil
			}
		}
		return false, nil
	})

	if err != nil {
		return nil, err
	}

	return txconflicts, nil
}

// The function detects conflicts in unconfirmed transactions list
// This is for case when some transaction output was used for 2 or more transactions input
// For building of a block we should use only one of them.
// Transaction can be used more 1 time in a block. But each time must be differeent output index
// TODO should work for different types of ransactions
func (u *unApprovedTransactions) DetectConflicts(txs []structures.Transaction) ([]structures.Transaction, []structures.Transaction, error) {
	return u.detectConflictsForCurrentcy(txs)
}

func (u *unApprovedTransactions) detectConflictsForCurrentcy(txs []structures.Transaction) ([]structures.Transaction, []structures.Transaction, error) {
	goodtransactions := []structures.Transaction{}
	conflicts := []structures.Transaction{}

	usedoutputs := map[string][]int{}

	for _, tx := range txs {
		used := false

		for _, txi := range tx.Vin {
			txinhax := hex.EncodeToString(txi.Txid)

			// check if this input was already used
			if vouts, ok := usedoutputs[txinhax]; ok {
				for _, vout := range vouts {
					if vout == txi.Vout {
						// used by other transaction!
						used = true
						break
					}
				}

				if !used {
					// it was not yet used. add to the list
					usedoutputs[txinhax] = append(usedoutputs[txinhax], txi.Vout)
				}
			} else {
				// this transaction is not yet in the map. add it
				usedoutputs[txinhax] = []int{txi.Vout}
			}

			if used {
				// add to conflicting transactions. we will have to delete them
				conflicts = append(conflicts, tx)
				break
			}
		}

		if !used {
			goodtransactions = append(goodtransactions, tx)
		}
	}

	return goodtransactions, conflicts, nil
}

// Is used for case when a block canceled. all transactions from a block are back to unapproved cache
func (u *unApprovedTransactions) AddFromCanceled(block *structures.Block) error {
	for _, tx := range block.Transactions {
		if !tx.IsCoinbaseTransfer() {
			err := u.Add(&tx)

			if err != nil {
				u.Logger.Trace.Printf("add tx %x error %s", tx.GetID(), err.Error())
			} else {
				u.Logger.Trace.Printf("UnAppr. add tx from canceled block. fine %x", tx.GetID())
			}
		}
	}

	return nil

}

// Remove all unapproced transactions from the pool
func (u *unApprovedTransactions) CleanUnapprovedCache() error {

	u.Logger.Trace.Println("Clean Unapproved Transactions cache: Prepare")

	utdb, err := u.DB.GetUnapprovedTransactionsObject()

	if err != nil {
		return err
	}
	err = utdb.TruncateDB()

	if err != nil {
		return err
	}
	if transactionsCacheEnable {
		u.lockCache()
		defer u.unlockCache()

		transactionsCache = make(map[string]structures.Transaction, 0)
	}

	return nil
}

// Find if there is transaction in a pool that updates given Reference
// can be used for some operations. INSERT can be based on a table create operation
// for now this is the only case when altid is really used
// Current TX can be specified. It is for case when we do verify and a TX is already in a pool
func (u *unApprovedTransactions) FindSQLReferenceTransaction(sqlUpdate structures.SQLUpdate) (txID []byte, err error) {
	// it i needed to go over all tranactions in cache and check each of them

	sqlUpdateMan, err := dbquery.NewSQLUpdateManager(sqlUpdate)

	if err != nil {
		return
	}

	// if not found, try to get alt ID
	altRefID, altCanBeReused, err := sqlUpdateMan.GetAlternativeRefID()

	if err != nil {
		return
	}

	var AlttxID []byte

	u.Logger.Trace.Printf("Search base TX in the pool for RefID %s and AltID %s", string(sqlUpdate.ReferenceID), string(altRefID))

	// this keeps list of transactions that were already used in other transations as a reference input
	transactionsReused := [][]byte{}

	// function tp process transaction in a loop
	err = u.forEachTransaction(func(tx *structures.Transaction) (bool, error) {
		if !tx.IsSQLCommand() {
			return false, nil
		}
		// RefID in a tarnsaction can not be empty

		//u.Logger.Trace.Printf("Check RefID %s in TX %x", string(tx.SQLCommand.ReferenceID), tx.GetID())

		if bytes.Compare(tx.SQLCommand.ReferenceID, sqlUpdate.ReferenceID) == 0 {
			// we found this refereence , check if input TX was not yet used as input in other tx
			if !u.helperCheckTXInList(tx.GetID(), transactionsReused) {
				//u.Logger.Trace.Printf("found TX %x", tx.GetID())
				txID = utils.CopyBytes(tx.GetID())
			}
		}

		if bytes.Compare(tx.SQLCommand.ReferenceID, altRefID) == 0 {

			// we found this refereence , check if input TX was not yet used as input in other tx
			if altCanBeReused || !u.helperCheckTXInList(tx.GetID(), transactionsReused) {
				AlttxID = tx.GetID()
				//u.Logger.Trace.Printf("found alt ID TX %x", tx.GetID())

				if altCanBeReused {
					//u.Logger.Trace.Printf("stop foreach %x", tx.GetID())
					// stop this loop. no sense to continue
					return true, nil
				}
			}
		}
		if len(tx.GetSQLBaseTX()) > 0 {
			transactionsReused = append(transactionsReused, tx.GetSQLBaseTX())
		}

		return false, nil
	})

	if len(txID) == 0 && len(AlttxID) > 0 {
		txID = AlttxID
	}

	return
}

// Find SQL TX based on specific TX
func (u *unApprovedTransactions) FindSQLBasedOnTransaction(txid []byte) (txIDs [][]byte, err error) {
	// it i needed to go over all tranactions in cache and check each of them

	txIDs = [][]byte{}
	u.Logger.Trace.Println("FindSQLBasedOnTransaction")

	err = u.forEachTransaction(func(tx *structures.Transaction) (bool, error) {
		if !tx.IsSQLCommand() {
			return false, nil
		}

		if bytes.Compare(tx.GetSQLBaseTX(), txid) == 0 {
			u.Logger.Trace.Printf("Check RefID %s in TX %x", string(tx.SQLCommand.ReferenceID), tx.GetID())

			txID := utils.CopyBytes(tx.GetID())
			txIDs = append(txIDs, txID)
		}

		return false, nil
	})

	return
}

// helper function to check if a TX is in the list of TXs
func (u *unApprovedTransactions) helperCheckTXInList(tx []byte, transactionsReused [][]byte) bool {
	for _, ttx := range transactionsReused {
		if bytes.Compare(tx, ttx) == 0 {
			return true
		}
	}
	return false
}
