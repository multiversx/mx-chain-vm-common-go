package mock

// DataTrieTrackerStub -
type DataTrieTrackerStub struct {
	ClearDataCachesCalled func()
	DirtyDataCalled       func() map[string][]byte
	RetrieveValueCalled   func(key []byte) ([]byte, uint32, error)
	SaveKeyValueCalled    func(key []byte, value []byte) error
}

// ClearDataCaches -
func (dtts *DataTrieTrackerStub) ClearDataCaches() {
	if dtts.ClearDataCachesCalled != nil {
		dtts.ClearDataCachesCalled()
	}
}

// DirtyData -
func (dtts *DataTrieTrackerStub) DirtyData() map[string][]byte {
	if dtts.DirtyDataCalled != nil {
		return dtts.DirtyDataCalled()
	}
	return nil
}

// RetrieveValue -
func (dtts *DataTrieTrackerStub) RetrieveValue(key []byte) ([]byte, uint32, error) {
	if dtts.RetrieveValueCalled != nil {
		return dtts.RetrieveValueCalled(key)
	}
	return nil, 0, nil
}

// SaveKeyValue -
func (dtts *DataTrieTrackerStub) SaveKeyValue(key []byte, value []byte) error {
	if dtts.SaveKeyValueCalled != nil {
		return dtts.SaveKeyValueCalled(key, value)
	}
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (dtts *DataTrieTrackerStub) IsInterfaceNil() bool {
	return dtts == nil
}
