package mock

import "github.com/multiversx/mx-chain-core-go/core"

// EnableEpochsHandlerStub -
type EnableEpochsHandlerStub struct {
	IsFlagEnabledInCurrentEpochCalled func(flag core.EnableEpochFlag) bool
}

// IsFlagEnabledInCurrentEpoch -
func (stub *EnableEpochsHandlerStub) IsFlagEnabledInCurrentEpoch(flag core.EnableEpochFlag) bool {
	if stub.IsFlagEnabledInCurrentEpochCalled != nil {
		return stub.IsFlagEnabledInCurrentEpochCalled(flag)
	}
	return false
}

// IsInterfaceNil -
func (stub *EnableEpochsHandlerStub) IsInterfaceNil() bool {
	return stub == nil
}
