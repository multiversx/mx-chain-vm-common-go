package mock

import (
	"github.com/multiversx/mx-chain-core-go/core"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

// DataTrieTrackerStub -
type DataTrieTrackerStub struct {
	ClearDataCachesCalled       func()
	DirtyDataCalled             func() map[string][]byte
	RetrieveValueCalled         func(key []byte) ([]byte, uint32, error)
	SaveKeyValueCalled          func(key []byte, value []byte) error
	MigrateDataTrieLeavesCalled func(oldVersion core.TrieNodeVersion, newVersion core.TrieNodeVersion, trieMigrator vmcommon.DataTrieMigrator) error
}

// ClearDataCaches -
func (dtts *DataTrieTrackerStub) ClearDataCaches() {
	dtts.ClearDataCachesCalled()
}

// DirtyData -
func (dtts *DataTrieTrackerStub) DirtyData() map[string][]byte {
	return dtts.DirtyDataCalled()
}

// RetrieveValue -
func (dtts *DataTrieTrackerStub) RetrieveValue(key []byte) ([]byte, uint32, error) {
	return dtts.RetrieveValueCalled(key)
}

// SaveKeyValue -
func (dtts *DataTrieTrackerStub) SaveKeyValue(key []byte, value []byte) error {
	return dtts.SaveKeyValueCalled(key, value)
}

// MigrateDataTrieLeaves -
func (dtts *DataTrieTrackerStub) MigrateDataTrieLeaves(oldVersion core.TrieNodeVersion, newVersion core.TrieNodeVersion, trieMigrator vmcommon.DataTrieMigrator) error {
	return dtts.MigrateDataTrieLeavesCalled(oldVersion, newVersion, trieMigrator)
}

// IsInterfaceNil returns true if there is no value under the interface
func (dtts *DataTrieTrackerStub) IsInterfaceNil() bool {
	return dtts == nil
}
