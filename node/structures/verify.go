package structures

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/gelembjuk/oursql/lib/utils"
)

// Check if all TXt currency outputs are only to given addresses
func CheckTXOutputsAreOnlyToGivenAddresses(tx *Transaction, PubKeyHashes [][]byte) error {

	for _, out := range tx.Vout {
		found := false

		for _, pubKeyHash := range PubKeyHashes {
			if bytes.Compare(pubKeyHash, out.PubKeyHash) == 0 {
				found = true
				break
			}
		}

		if !found {
			addr, err := utils.PubKeyHashToAddres(out.PubKeyHash)

			if err != nil {
				return err
			}
			return errors.New(fmt.Sprintf("Out address %s is not found in the given list", addr))
		}
	}

	return nil
}

func CheckTXOutputValueToAddress(tx *Transaction, PubKeyHash []byte, amount float64) error {

	total := float64(0)
	hasamount := false

	// must work corrrectly when sending to himself

	for _, out := range tx.Vout {
		if bytes.Compare(out.PubKeyHash, PubKeyHash) == 0 {
			total = total + out.Value

			if out.Value == amount {
				hasamount = true
			}
		}
	}

	if total != amount && !hasamount {
		addr, err := utils.PubKeyHashToAddres(PubKeyHash)

		if err != nil {
			return err
		}
		return errors.New(fmt.Sprintf("TX output value for %s is %f, not %f as expected", addr, total, amount))
	}

	return nil
}
