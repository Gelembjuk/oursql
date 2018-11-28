package transactions

import (
	"bytes"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
)

// This structure is used to manage association of data rows and transactions
// where a row was last changes in a blockchain
// The idea of this index is to help quickly to find where a row was changed before
// to find a transaction of previous change and understand if there is no other transation that is
// based on same previous TX

type rowsToTransactions struct {
	DB     database.DBManager
	Logger *utils.LoggerMan
}

// Create tx index object to use in this structure
func (dr rowsToTransactions) getIndexManager() *transactionsIndex {
	return newTransactionIndex(dr.DB, dr.Logger)
}

// Block Removed Fom Main branch
// We change index of row references and transactions
func (dr rowsToTransactions) UpdateOnBlockCancel(block *structures.Block) error {
	drdb, err := dr.DB.GetDataReferencesObject()

	if err != nil {
		return err
	}

	dr.Logger.Trace.Printf("Data References on block remove %x", block.Hash)

	for _, tx := range block.Transactions {
		dr.Logger.Trace.Printf("Data References check tx %x", tx.GetID())

		// there are 2 options. Previous TX can be upadte of this row or it can be table create
		// we can update this reference or delete it

		if !tx.IsSQLCommand() {
			// nothing to do
			continue
		}

		if len(tx.SQLCommand.ReferenceID) == 0 {
			// no any reference here
			continue
		}
		dr.Logger.Trace.Printf("TX %x , refID %s", tx.GetID(), string(tx.SQLCommand.ReferenceID))
		curTX, err := drdb.GetTXForRefID(tx.SQLCommand.ReferenceID)

		if err != nil {
			// in which cases can it be? it should not happen
			return err
		}

		if curTX == nil {
			// there is no reference for this row. skipping it. nothing to do
			continue
		}
		if bytes.Compare(curTX, tx.GetID()) != 0 {
			// reference is for other TX. this should not happen
			// TODO if this happens , it is needed to find why and fix something
			continue
		}
		if len(tx.GetSQLBaseTX()) == 0 {
			drdb.DeleteRefID(tx.SQLCommand.ReferenceID)
			// delete and this is done
			continue
		}
		// get previous TX to understand what is the type. Maybe that
		txPrev, err := dr.getIndexManager().GetTransaction(tx.GetSQLBaseTX(), []byte{})

		if err != nil {
			return err
		}

		if txPrev != nil {
			if bytes.Compare(tx.SQLCommand.ReferenceID, txPrev.SQLCommand.ReferenceID) == 0 {
				// only if previous TX we worked with same row
				drdb.SetTXForRefID(tx.SQLCommand.ReferenceID, txPrev.GetID())
			}
		}
		// in other cases just delete it
		drdb.DeleteRefID(tx.SQLCommand.ReferenceID)
	}

	return nil
}

// Block Added To Main branch
// We change index of row references and transactions
func (dr rowsToTransactions) UpdateOnBlockAdd(block *structures.Block) error {
	drdb, err := dr.DB.GetDataReferencesObject()

	if err != nil {
		return err
	}

	//dr.Logger.Trace.Printf("Data References on block add %x", block.Hash)

	for _, tx := range block.Transactions {
		//dr.Logger.Trace.Printf("Data References check tx %x", tx.GetID())

		if !tx.IsSQLCommand() {
			// nothing to do
			continue
		}

		if len(tx.SQLCommand.ReferenceID) == 0 {
			// no any reference here
			dr.Logger.Trace.Printf("NO Reference for  %s", string(tx.GetSQLQuery()))
			continue
		}
		//dr.Logger.Trace.Printf("TX %x , refID %s", tx.GetID(), string(tx.SQLCommand.ReferenceID))

		// we set new association
		err = drdb.SetTXForRefID(tx.SQLCommand.ReferenceID, tx.GetID())

		if err != nil {
			return err
		}
	}

	return nil
}

func (dr rowsToTransactions) GetTXForRefID(RefID []byte) (txID []byte, err error) {
	drdb, err := dr.DB.GetDataReferencesObject()

	if err != nil {
		return
	}

	return drdb.GetTXForRefID(RefID)
}
