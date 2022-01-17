package builtInFunctions

import (
	mockvm "github.com/ElrondNetwork/elrond-vm-common/mock"
)

var marshallerMock = &mockvm.MarshalizerMock{}

func createBaseAccountFreezerArgs() BaseAccountFreezerArgs {
	blockChainHook := &mockvm.BlockChainEpochHookStub{
		CurrentEpochCalled: func() uint32 {
			return 1000
		},
	}

	return BaseAccountFreezerArgs{
		BlockChainHook: blockChainHook,
		Marshaller:     marshallerMock,
		EpochNotifier:  &mockvm.EpochNotifierStub{},
		FuncGasCost:    100000,
	}
}
