package dataTrieMigrator

import (
	"github.com/multiversx/mx-chain-core-go/core"
)

// ArgsNewDataTrieMigrator is the arguments structure for the new dataTrieMigrator component
type ArgsNewDataTrieMigrator struct {
	GasProvided uint64
	DataTrieGasCost
}

// DataTrieGasCost contains the gas costs for the data trie load and store operations
type DataTrieGasCost struct {
	TrieLoadPerNode  uint64
	TrieStorePerNode uint64
}

type dataTrieMigrator struct {
	gasRemaining       uint64
	trieLoadCost       uint64
	trieMigrateCost    uint64
	leavesToBeMigrated []core.TrieData
}

// NewDataTrieMigrator creates a new dataTrieMigrator component
func NewDataTrieMigrator(args ArgsNewDataTrieMigrator) *dataTrieMigrator {
	return &dataTrieMigrator{
		gasRemaining:       args.GasProvided,
		trieLoadCost:       args.TrieLoadPerNode,
		trieMigrateCost:    args.TrieStorePerNode,
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

// GetLeavesToBeMigrated returns the list of leaves to be migrated
func (dtm *dataTrieMigrator) GetLeavesToBeMigrated() []core.TrieData {
	return dtm.leavesToBeMigrated
}

// GetGasRemaining returns the remaining gas
func (dtm *dataTrieMigrator) GetGasRemaining() uint64 {
	return dtm.gasRemaining
}

// IsInterfaceNil returns true if there is no value under the interface
func (dtm *dataTrieMigrator) IsInterfaceNil() bool {
	return dtm == nil
}
