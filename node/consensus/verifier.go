package consensus

/*
* This will have all code related to verify of transactions (and possible blocks)
* on level of consensus
 */

import (
	"github.com/gelembjuk/oursql/lib"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/dbquery"
)

type verifyManager struct {
	logger             *utils.LoggerMan
	previousBlockHeigh int
	config             *ConsensusConfig
}

// check if this pubkey can execute this query
func (vm verifyManager) CheckExecutePermissions(qp *dbquery.QueryParsed, pubKey []byte) (bool, error) {
	// check sonsensus rules
	if !qp.IsUpdate() {
		return true, nil
	}

	hasCustom, allow, err := vm.checkExecutePermissionsAsTable(qp, pubKey)

	if err != nil {
		return false, err
	}

	if hasCustom {
		return allow, nil
	}

	if qp.Structure.GetKind() == lib.QueryKindCreate {
		if !vm.config.AllowTableCreate {
			return false, nil
		}
	}

	if qp.Structure.GetKind() == lib.QueryKindDrop {
		if !vm.config.AllowTableDrop {
			return false, nil
		}
	}

	if qp.Structure.GetKind() == lib.QueryKindDelete {
		if !vm.config.AllowRowDelete {
			return false, nil
		}
	}

	return true, nil
}

// check custom rule for the table about permissions
func (vm verifyManager) checkExecutePermissionsAsTable(qp *dbquery.QueryParsed, pubKey []byte) (hasCustom bool, allow bool, err error) {
	hasCustom = false

	t := vm.config.getTableCustomConfig(qp)

	if t != nil {
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
func (vm verifyManager) CheckQueryNeedsPayment(qp *dbquery.QueryParsed) (float64, error) {

	// check there is custom rule for this table
	t := vm.config.getTableCustomConfig(qp)

	var trcost *ConsensusConfigCost

	if t != nil {
		trcost = &t.TransactionCost
	} else {
		trcost = &vm.config.TransactionCost
	}

	// check if current operation has a price
	if qp.Structure.GetKind() == lib.QueryKindDelete && trcost.RowDelete > 0 {

		return trcost.RowDelete, nil
	}

	if qp.Structure.GetKind() == lib.QueryKindInsert && trcost.RowInsert > 0 {

		return trcost.RowInsert, nil
	}

	if qp.Structure.GetKind() == lib.QueryKindUpdate && trcost.RowUpdate > 0 {
		return trcost.RowUpdate, nil
	}

	if qp.Structure.GetKind() == lib.QueryKindCreate && trcost.TableCreate > 0 {

		return trcost.TableCreate, nil
	}

	if trcost.Default > 0 {
		return trcost.Default, nil
	}

	return 0, nil
}
