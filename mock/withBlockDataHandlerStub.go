package mock

import vmcommon "github.com/multiversx/mx-chain-vm-common-go"

// BlockchainDataProviderStub -
type BlockchainDataProviderStub struct {
	SetBlockDataHandlerCalled func(handler vmcommon.BlockchainDataHook) error
	CurrentRoundCalled        func() uint64
}

// SetBlockchainHook -
func (w *BlockchainDataProviderStub) SetBlockchainHook(handler vmcommon.BlockchainDataHook) error {
	if w.SetBlockDataHandlerCalled != nil {
		return w.SetBlockDataHandlerCalled(handler)
	}
	return nil
}

// CurrentRound -
func (w *BlockchainDataProviderStub) CurrentRound() uint64 {
	if w.CurrentRoundCalled != nil {
		return w.CurrentRoundCalled()
	}
	return 0
}

// IsInterfaceNil -
func (w *BlockchainDataProviderStub) IsInterfaceNil() bool {
	return w == nil
}
