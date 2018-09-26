package transactions

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"log"
	"sort"

	"github.com/gelembjuk/oursql/lib/remoteclient"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/blockchain"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
)

// unspentTransactions represents UTXO set
type unspentTransactions struct {
	DB     database.DBManager
	Logger *utils.LoggerMan
}

// Create transaction index object
func (u unspentTransactions) newTransactionIndex() *transactionsIndex {
	return newTransactionIndex(u.DB, u.Logger)
}

// Serialize. We need this to store data in DB in bytes
func (u unspentTransactions) serializeOutputs(outs []structures.TXOutputIndependent) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

/*
* Deserialize data from bytes loaded fom DB
 */
func (u unspentTransactions) deserializeOutputs(data []byte) ([]structures.TXOutputIndependent, error) {
	var outputs []structures.TXOutputIndependent

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	if err != nil {
		return nil, err
		log.Panic(err)
	}

	return outputs, nil
}

/*
* Calculates address balance using the cache of unspent transactions outputs
 */
func (u unspentTransactions) GetAddressBalance(address string) (float64, error) {
	if address == "" {
		return 0, errors.New("Address is missed")
	}
	w := remoteclient.Wallet{}

	if !w.ValidateAddress(address) {
		return 0, errors.New("Address is not valid")
	}

	balance := float64(0)

	UnspentTXs, err2 := u.GetunspentTransactionsOutputs(address)

	if err2 != nil {
		return 0, err2
	}

	for _, out := range UnspentTXs {
		balance += out.Value
	}
	return balance, nil
}

// CGet input value. Input is unspent TX output
func (u unspentTransactions) GetInputValue(input structures.TXCurrencyInput) (float64, error) {

	uodb, err := u.DB.GetUnspentOutputsObject()

	if err != nil {
		return 0, err
	}

	outsBytes, err := uodb.GetDataForTransaction(input.Txid)

	if err != nil {
		u.Logger.Trace.Printf("Data reading error: %s", err.Error())
		return 0, err
	}

	if outsBytes == nil {
		return 0, NewTXNotFoundUOTError("Input TX is not found in unspent outputs")
	}

	outs, err := u.deserializeOutputs(outsBytes)

	if err != nil {
		return 0, err
	}

	for _, o := range outs {
		if o.OIndex == input.Vout {
			return o.Value, nil
		}
	}

	return 0, errors.New("Output index is not found in unspent outputs")
}

// Choose inputs for new transaction
func (u unspentTransactions) ChooseSpendableOutputs(pubKeyHash []byte, amount float64,
	pendinguse []structures.TXCurrencyInput) (float64, []structures.TXOutputIndependent, error) {

	uodb, err := u.DB.GetUnspentOutputsObject()

	if err != nil {
		return 0, nil, err
	}

	unspentOutputs := []structures.TXOutputIndependent{}
	accumulated := float64(0)

	err = uodb.ForEach(func(txID, txData []byte) error {

		outs, err := u.deserializeOutputs(txData)

		if err != nil {
			return err
		}

		for _, out := range outs {
			if out.IsLockedWithKey(pubKeyHash) {
				// check if this output is not used in some pending transaction
				used := false
				for _, pin := range pendinguse {
					if bytes.Compare(pin.Txid, out.TXID) == 0 &&
						pin.Vout == out.OIndex {
						used = true
						break
					}
				}
				if used {
					continue
				}
				accumulated += out.Value
				unspentOutputs = append(unspentOutputs, out)
			}
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}

	if accumulated >= amount {
		// choose longest number of outputs to spent. it must be outs with smallest amounts
		sort.Sort(structures.TXOutputIndependentList(unspentOutputs))

		accumulated = 0
		uo := []structures.TXOutputIndependent{}

		for _, out := range unspentOutputs {

			accumulated += out.Value
			uo = append(uo, out)

			if accumulated >= amount {
				break
			}
		}

		unspentOutputs = uo
	}

	return accumulated, unspentOutputs, nil
}

// execute callback function for every record in unspent outputs
func (u unspentTransactions) forEachUnspentOutput(address string, callback UnspentTransactionOutputCallbackInterface) error {
	pubKeyHash, err := utils.AddresToPubKeyHash(address)

	if err != nil {
		return err
	}
	uodb, err := u.DB.GetUnspentOutputsObject()

	if err != nil {
		return err
	}

	err = uodb.ForEach(func(txID, txData []byte) error {
		outs, err := u.deserializeOutputs(txData)

		if err != nil {
			return err
		}

		for _, out := range outs {
			if out.IsLockedWithKey(pubKeyHash) {
				var fromaddr string

				if len(out.SendPubKeyHash) > 0 {
					fromaddr, _ = utils.PubKeyHashToAddres(out.SendPubKeyHash)
				} else {
					fromaddr = "Coin base"
				}
				err := callback(fromaddr, out.Value, out.TXID, out.OIndex, out.IsBase)

				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Returns list of unspent transactions outputs for address
func (u unspentTransactions) GetunspentTransactionsOutputs(address string) ([]structures.TXOutputIndependent, error) {
	if address == "" {
		return nil, errors.New("Address is missed")
	}
	w := remoteclient.Wallet{}

	if !w.ValidateAddress(address) {
		return nil, errors.New("Address is not valid")
	}
	pubKeyHash, err := utils.AddresToPubKeyHash(address)

	if err != nil {
		return nil, err
	}
	uodb, err := u.DB.GetUnspentOutputsObject()

	if err != nil {
		return nil, err
	}

	UTXOs := []structures.TXOutputIndependent{}
	//u.Logger.Trace.Printf("findoutputs for  %s", address)
	err = uodb.ForEach(func(txID, txData []byte) error {
		outs, err := u.deserializeOutputs(txData)

		if err != nil {
			return err
		}

		for _, out := range outs {
			//u.Logger.Trace.Printf("output %x %d", out.TXID, out.OIndex)
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return UTXOs, nil
}

// Returns total number of unspent transactions in a cache.
func (u unspentTransactions) CountTransactions() (int, error) {
	uodb, err := u.DB.GetUnspentOutputsObject()

	if err != nil {
		return 0, err
	}

	return uodb.GetCount()
}

// Returns toal number of transactions outputs in a cache

func (u unspentTransactions) CountUnspentOutputs() (int, error) {
	uodb, err := u.DB.GetUnspentOutputsObject()

	if err != nil {
		return 0, err
	}

	counter := 0

	err = uodb.ForEach(func(txID, txData []byte) error {
		outs, err := u.deserializeOutputs(txData)

		if err != nil {
			return err
		}

		counter += len(outs)
		return nil
	})
	if err != nil {
		return 0, err
	}

	return counter, nil
}

// Rebuilds the DB of unspent transactions
// NOTE . We don't really need this. Normal code should work without reindexing.
// TODO to remove this function in future
func (u unspentTransactions) Reindex() (int, error) {
	u.Logger.Trace.Println("Reindex UTXO: Prepare")

	uodb, err := u.DB.GetUnspentOutputsObject()

	if err != nil {
		return 0, err
	}

	err = uodb.TruncateDB()

	if err != nil {
		return 0, err
	}

	u.Logger.Trace.Println("Reindex UTXO: Prepare done")

	UTXO, err := u.FindunspentTransactions()

	if err != nil {
		return 0, err
	}

	u.Logger.Trace.Println("Reindex UTXO: Store records")

	for txID, outs := range UTXO {
		u.Logger.Trace.Printf("Reindex UTXO: Save %s %d", txID, len(outs))

		key, err := hex.DecodeString(txID)
		if err != nil {
			return 0, err
		}

		outsData, err := u.serializeOutputs(outs)

		if err != nil {
			return 0, err
		}

		err = uodb.PutDataForTransaction(key, outsData)

		if err != nil {
			return 0, err
		}
	}

	u.Logger.Trace.Println("Reindex UTXO: Done. Return counts")
	return u.CountUnspentOutputs()
}

// Returns full list of unspent transactions outputs
// Iterates over full blockchain
// TODO this will not work for big blockchain. It keeps data in memory

func (u unspentTransactions) FindunspentTransactions() (map[string][]structures.TXOutputIndependent, error) {
	UTXO := make(map[string][]structures.TXOutputIndependent)
	spentTXOs := make(map[string][]int)

	bci, err := blockchain.NewBlockchainIterator(u.DB)

	if err != nil {
		return nil, err
	}

	u.Logger.Trace.Println("Get All UTXO: Start")

	for {
		block, _ := bci.Next()

		for j := len(block.Transactions) - 1; j >= 0; j-- {
			tx := &block.Transactions[j]

			txID := hex.EncodeToString(tx.GetID())

			sender := []byte{}

			if !tx.IsCoinbaseTransfer() {
				sender, _ = utils.HashPubKey(tx.ByPubKey)
			}

			var spent bool

			for outIdx, out := range tx.Vout {
				// Was the output spent?
				spent = false

				if list, ok := spentTXOs[txID]; ok {

					for _, spentOutIdx := range list {

						if spentOutIdx == outIdx {
							// this output of the transaction was already spent
							// go to next output of this transaction
							spent = true
							break
						}
					}
				}
				if spent {
					continue
				}
				// add to unspent

				if _, ok := UTXO[txID]; !ok {
					UTXO[txID] = []structures.TXOutputIndependent{}
				}
				outs := UTXO[txID]

				oute := structures.TXOutputIndependent{}
				oute.LoadFromSimple(out, tx.GetID(), outIdx, sender, tx.IsCoinbaseTransfer(), block.Hash)

				outs = append(outs, oute)
				UTXO[txID] = outs
			}

			if tx.IsCoinbaseTransfer() {
				continue
			}
			for _, in := range tx.Vin {
				inTxID := hex.EncodeToString(in.Txid)

				if _, ok := spentTXOs[inTxID]; !ok {
					spentTXOs[inTxID] = []int{}
				}
				spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)

			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	u.Logger.Trace.Printf("Get All UTXO: Return %d records", len(UTXO))
	return UTXO, nil
}

/*
* New Block added
* Input of all tranactions are removed from unspent
* OUtput of all transactions are added to unspent
* Update the UTXO set with transactions from the Block
* The Block is considered to be the tip of a blockchain
 */
func (u unspentTransactions) UpdateOnBlockAdd(block *structures.Block) error {
	uodb, err := u.DB.GetUnspentOutputsObject()

	if err != nil {
		u.Logger.Trace.Printf("UTXO, db err %e", err.Error())
		return err
	}

	u.Logger.Trace.Printf("UPdate UTXO on block add %x", block.Hash)

	for _, tx := range block.Transactions {
		//u.Logger.Trace.Printf("UpdateOnBlockAdd check tx %x", tx.GetID())

		sender := []byte{}

		if !tx.IsCoinbaseTransfer() {
			sender, _ = utils.HashPubKey(tx.ByPubKey)

			for _, vin := range tx.Vin {

				outsBytes, err := uodb.GetDataForTransaction(vin.Txid)

				if err != nil {
					return err
				}

				if outsBytes == nil {
					u.Logger.Trace.Printf("UpdateOnBlockAdd in tx is not found %x", vin.Txid)
					continue
				}

				outs, err := u.deserializeOutputs(outsBytes)

				if err != nil {
					return err
				}

				updatedOuts := []structures.TXOutputIndependent{}

				for _, out := range outs {
					if out.OIndex != vin.Vout {
						updatedOuts = append(updatedOuts, out)
					}
				}

				if len(updatedOuts) == 0 {
					err = uodb.DeleteDataForTransaction(vin.Txid)
				} else {
					d, err := u.serializeOutputs(updatedOuts)
					if err != nil {
						return err
					}
					err = uodb.PutDataForTransaction(vin.Txid, d)

					if err != nil {
						return err
					}
				}

				if err != nil {
					return err
				}
			}
		}
		newOutputs := []structures.TXOutputIndependent{}

		for outInd, out := range tx.Vout {
			no := structures.TXOutputIndependent{}
			no.LoadFromSimple(out, tx.ID, outInd, sender, tx.IsCoinbaseTransfer(), block.Hash)
			newOutputs = append(newOutputs, no)
		}

		d, err := u.serializeOutputs(newOutputs)

		if err != nil {
			return err
		}
		//u.Logger.Trace.Printf("BA tx save as unspent %x %d outputs", tx.ID, len(newOutputs))
		err = uodb.PutDataForTransaction(tx.ID, d)

		if err != nil {
			return err
		}
	}

	return nil
}

// This is executed when a block is canceled.
// All input transactions must be return to "unspent"
// And all output must be deleted from "unspent"
func (u unspentTransactions) UpdateOnBlockCancel(block *structures.Block) error {
	uodb, err := u.DB.GetUnspentOutputsObject()

	if err != nil {
		return err
	}

	u.Logger.Trace.Printf("block cancel at unspent %x , prev hash %x", block.Hash, block.PrevBlockHash) //REM

	for _, tx := range block.Transactions {
		u.Logger.Trace.Printf("BC check tx %x", tx.GetID()) //REM

		// delete this transaction from list of unspent
		uodb.DeleteDataForTransaction(tx.GetID())

		if tx.IsCoinbaseTransfer() {
			continue
		}
		sender, _ := utils.HashPubKey(tx.ByPubKey)
		// all input outputs must be added back to unspent
		// but only if inputs are in current BC
		for _, vin := range tx.Vin {
			// when we execute cancel, current top can be already changed. we use this block hash as a top
			// to find this TX
			txi, spending, blockHash, err := u.newTransactionIndex().GetCurrencyTransactionAllInfo(vin.Txid, block.PrevBlockHash)

			//u.Logger.Trace.Printf("input tx find input %x", vin.Txid) //REM

			if err != nil {
				//u.Logger.Trace.Printf("error finding tx %x %s", tx.ID, err.Error()) //REM
				return err
			}

			if txi == nil {
				// TX is not found in current BC . no sense to add it to unspent
				u.Logger.Trace.Printf("tx not found in current BC") //REM
				break
			}

			//u.Logger.Trace.Printf("found tx in block %x", blockHash)   //REM
			//u.Logger.Trace.Printf("spendings count %d", len(spending)) //REM
			//u.Logger.Trace.Printf("spendings count %s", spending)      //REM

			UnspentOuts := []structures.TXOutputIndependent{}

			for outInd, out := range txi.Vout {
				spent := false

				for _, so := range spending {
					if so.OutInd == outInd {
						spent = true
						break
					}
				}
				if !spent {
					no := structures.TXOutputIndependent{}
					no.LoadFromSimple(out, txi.ID, outInd, sender, tx.IsCoinbaseTransfer(), blockHash)

					UnspentOuts = append(UnspentOuts, no)
				}
			}
			//u.Logger.Trace.Printf("BC tx save as unspent %x %d outputs", vin.Txid, len(UnspentOuts))

			if len(UnspentOuts) > 0 {
				txData, err := u.serializeOutputs(UnspentOuts)

				if err != nil {
					return err
				}

				err = uodb.PutDataForTransaction(vin.Txid, txData)

				if err != nil {
					return err
				}
			} else {
				uodb.DeleteDataForTransaction(vin.Txid)
			}

		}
	}

	return nil
}

// Find inputs for new structures. Receives list of pending inputs used in other
// not yet confirmed transactions
// Returns list of inputs prepared. Even if less then requested
// Returns previous transactions. It later will be used to prepare data to sign
func (u unspentTransactions) GetNewTransactionInputs(PubKey []byte, to string, amount float64,
	pendinguse []structures.TXCurrencyInput) ([]structures.TXCurrencyInput, map[string]*structures.Transaction, float64, error) {

	localError := func(err error) ([]structures.TXCurrencyInput, map[string]*structures.Transaction, float64, error) {
		return nil, nil, 0, err
	}

	inputs := []structures.TXCurrencyInput{}

	pubKeyHash, _ := utils.HashPubKey(PubKey)
	totalamount, validOutputs, err := u.ChooseSpendableOutputs(pubKeyHash, amount, pendinguse)

	if err != nil {
		return localError(err)
	}

	bcMan, err := blockchain.NewBlockchainManager(u.DB, u.Logger)

	if err != nil {
		return localError(err)
	}
	// here we don't calculate is total amount is good or no.
	// later we will add unconfirmed transactions if no enough funds

	// build list of previous transactions
	prevTXs := make(map[string]*structures.Transaction)

	// Build a list of inputs
	for _, out := range validOutputs {
		input := structures.TXCurrencyInput{out.TXID, out.OIndex}
		inputs = append(inputs, input)

		prevTX, err := bcMan.GetTransactionFromBlock(out.TXID, out.BlockHash)

		if err != nil {
			return localError(err)
		}

		prevTXs[hex.EncodeToString(prevTX.GetID())] = prevTX
	}
	return inputs, prevTXs, totalamount, nil
}

// Returns previous transactions. It later will be used to prepare data to sign
func (u unspentTransactions) ExtendNewTransactionInputs(PubKey []byte, amount, totalamount float64,
	inputs []structures.TXCurrencyInput, prevTXs map[string]*structures.Transaction,
	pendingoutputs []*structures.TXOutputIndependent) ([]structures.TXCurrencyInput, map[string]*structures.Transaction, float64, error) {

	// Build a list of inputs
	for _, out := range pendingoutputs {
		input := structures.TXCurrencyInput{out.TXID, out.OIndex}
		inputs = append(inputs, input)

		prevTX, err := structures.DeserializeTransaction(out.BlockHash) // here we have transaction serialised, not block hash

		if err != nil {
			return inputs, prevTXs, totalamount, err
		}

		prevTXs[hex.EncodeToString(prevTX.GetID())] = prevTX

		totalamount += out.Value

		if totalamount >= amount {
			break
		}
	}
	return inputs, prevTXs, totalamount, nil
}

// Verifies which transactions outputs are not yet spent.
// Returns list of inputs that are not found in list of unspent outputs
func (u unspentTransactions) VerifyTransactionsOutputsAreNotSpent(txilist []structures.TXCurrencyInput) (map[int]structures.TXCurrencyInput, map[int]*structures.Transaction, error) {
	localError := func(err error) (map[int]structures.TXCurrencyInput, map[int]*structures.Transaction, error) {
		return nil, nil, err
	}
	// list of full input transactions. it can be used to verify signature later
	inputTX := map[int]*structures.Transaction{}

	notFoundInputs := make(map[int]structures.TXCurrencyInput)

	uodb, err := u.DB.GetUnspentOutputsObject()

	if err != nil {
		return localError(err)
	}

	bcMan, err := blockchain.NewBlockchainManager(u.DB, u.Logger)

	if err != nil {
		return localError(err)
	}

	for txiInd, txi := range txilist {
		txdata, err := uodb.GetDataForTransaction(txi.Txid)

		if err != nil {
			return localError(err)
		}

		if txdata == nil {
			// not found
			inputTX[txiInd] = nil
			notFoundInputs[txiInd] = txi
			continue
		}
		exists := false
		blockHash := []byte{}

		outs, err := u.deserializeOutputs(txdata)

		if err != nil {
			return localError(err)
		}

		for _, out := range outs {
			if out.OIndex == txi.Vout {
				exists = true
				blockHash = out.BlockHash
				break
			}
		}

		if !exists {
			notFoundInputs[txiInd] = txi
			inputTX[txiInd] = nil
		} else {
			// find this TX and get full info about it
			prevTX, err := bcMan.GetTransactionFromBlock(txi.Txid, blockHash)

			if err != nil {
				return localError(err)
			}
			inputTX[txiInd] = prevTX
		}
	}
	return notFoundInputs, inputTX, nil
}
