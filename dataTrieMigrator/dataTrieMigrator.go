package dataTrieMigrator

import (
	"github.com/multiversx/mx-chain-core-go/core"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type dataTrieMigrator struct {
	gasRemaining    uint64
	trieLoadCost    uint64
	trieMigrateCost uint64

	leavesToBeMigrated []core.TrieData
}

// NewDataTrieMigrator creates a new dataTrieMigrator component
func NewDataTrieMigrator(gasProvided uint64, builtInCost vmcommon.BuiltInCost) *dataTrieMigrator {
	return &dataTrieMigrator{
		gasRemaining:    gasProvided,
		trieLoadCost:    builtInCost.TrieLoad,
		trieMigrateCost: builtInCost.TrieStore,

		leavesToBeMigrated: make([]core.TrieData, 0),
	}
}

// ConsumeStorageLoadGas consumes gas for loading a trie node. It returns true if there is enough
// gas remaining for another trie node load.
func (dtm *dataTrieMigrator) ConsumeStorageLoadGas() bool {
	if dtm.gasRemaining < dtm.trieLoadCost {
		return false
	}

	dtm.gasRemaining -= dtm.trieLoadCost

	return dtm.gasRemaining > dtm.trieLoadCost
}

// AddLeafToMigrationQueue will add the given data to the list of leaves to be migrated.
// It returns true if there is enough gas remaining for another trie node migration.
func (dtm *dataTrieMigrator) AddLeafToMigrationQueue(leafData core.TrieData, newLeafVersion core.TrieNodeVersion) (bool, error) {
	if dtm.gasRemaining < dtm.trieMigrateCost {
		return false, nil
	}

	if newLeafVersion == core.AutoBalanceEnabled {
		dtm.prepareDataForMigrationToAutoBalance(leafData)
	}

	return dtm.gasRemaining > dtm.trieMigrateCost, nil
}

func (dtm *dataTrieMigrator) prepareDataForMigrationToAutoBalance(leafData core.TrieData) {
	if leafData.Version != core.NotSpecified {
		return
	}

	dtm.gasRemaining -= dtm.trieMigrateCost

	dtm.leavesToBeMigrated = append(dtm.leavesToBeMigrated, leafData)
}

// MigrateCollectedLeaves will migrate the collected leaves to the specified version in the given account's data trie.
func (dtm *dataTrieMigrator) MigrateCollectedLeaves(acntDst vmcommon.UserAccountHandler) error {
	for _, data := range dtm.leavesToBeMigrated {
		err := acntDst.AccountDataHandler().SaveTrieData(data)
		if err != nil {
			return err
		}
	}

	dtm.leavesToBeMigrated = make([]core.TrieData, 0)
	return nil
}

// GetGasRemaining returns the remaining gas
func (dtm *dataTrieMigrator) GetGasRemaining() uint64 {
	return dtm.gasRemaining
}
