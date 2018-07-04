package blockchain

import (
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/database"
	"github.com/gelembjuk/oursql/node/structures"
)

// BlockchainIterator is used to iterate over blockchain blocks
type BlockchainIterator struct {
	currentHash []byte
	DB          database.DBManager
}

// Creates new Blockchain Iterator . Can be used to do something with blockchain from outside

func NewBlockchainIterator(DB database.DBManager) (*BlockchainIterator, error) {

	bcdb, err := DB.GetBlockchainObject()

	if err != nil {
		return nil, err
	}

	starttip, err := bcdb.GetTopHash()

	if err != nil {
		return nil, err
	}

	return &BlockchainIterator{starttip, DB}, nil
}

// Creates new Blockchain Iterator from given block hash. Can be used to do something with blockchain from outside
//
func NewBlockchainIteratorFrom(DB database.DBManager, startHash []byte) (*BlockchainIterator, error) {
	return &BlockchainIterator{startHash, DB}, nil
}

// Next returns next block starting from the tip
func (i *BlockchainIterator) Next() (*structures.Block, error) {
	var block *structures.Block

	bcdb, err := i.DB.GetBlockchainObject()

	if err != nil {
		return nil, err
	}
	//fmt.Printf("request block %x", i.currentHash)
	encodedBlock, err := bcdb.GetBlock(i.currentHash)

	if err != nil {
		return nil, err
	}

	block, err = structures.NewBlockFromBytes(encodedBlock)

	if err != nil {
		return nil, err
	}

	i.currentHash = block.PrevBlockHash

	return block, nil
}

// Returns history of transactions for given address
func (i *BlockchainIterator) GetAddressHistory(pubKeyHash []byte, address string) ([]structures.TransactionsHistory, error) {
	result := []structures.TransactionsHistory{}

	for {
		block, _ := i.Next()

		for _, tx := range block.Transactions {

			if !tx.IsCurrencyTransfer() {
				// skip non currency transactions
				continue
			}

			income := float64(0)

			spent := false
			spentaddress, _ := utils.PubKeyToAddres(tx.ByPubKey)

			if tx.CreatedByPubKeyHash(pubKeyHash) {
				spent = true
			}

			if spent {
				// find how many spent , part of out can be exchange to same address

				spentvalue := float64(0)
				totalvalue := float64(0) // we need to know total if wallet sent to himself

				destaddress := ""

				// we agree that there can be only one destination in transaction. we don't support scripts
				for _, out := range tx.Vout {
					if !out.IsLockedWithKey(pubKeyHash) {
						spentvalue += out.Value
						destaddress, _ = utils.PubKeyHashToAddres(out.PubKeyHash)
					}
				}

				if spentvalue > 0 {
					result = append(result, structures.TransactionsHistory{false, tx.ID, destaddress, spentvalue})
				} else {
					// spent to himself. this should not be usual case
					result = append(result, structures.TransactionsHistory{false, tx.ID, address, totalvalue})
					result = append(result, structures.TransactionsHistory{true, tx.ID, address, totalvalue})
				}
			} else if tx.IsCoinbaseTransfer() {

				if tx.Vout[0].IsLockedWithKey(pubKeyHash) {
					spentaddress = "Coin base"
					income = tx.Vout[0].Value
				}
			} else {

				for _, out := range tx.Vout {

					if out.IsLockedWithKey(pubKeyHash) {
						income += out.Value
					}
				}
			}

			if income > 0 {
				result = append(result, structures.TransactionsHistory{true, tx.ID, spentaddress, income})
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return result, nil
}
