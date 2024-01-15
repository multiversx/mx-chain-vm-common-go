package mock

import vmcommon "github.com/multiversx/mx-chain-vm-common-go"

// WithBlockDataHandlerStub -
type WithBlockDataHandlerStub struct {
	SetBlockDataHandlerCalled func(handler vmcommon.BlockDataHandler) error
	CurrentRoundCalled        func() (uint64, error)
}

// SetBlockDataHandler -
func (w *WithBlockDataHandlerStub) SetBlockDataHandler(handler vmcommon.BlockDataHandler) error {
	if w.SetBlockDataHandlerCalled != nil {
		return w.SetBlockDataHandlerCalled(handler)
	}
	return nil
}

// CurrentRound -
func (w *WithBlockDataHandlerStub) CurrentRound() (uint64, error) {
	if w.CurrentRoundCalled != nil {
		return w.CurrentRoundCalled()
	}
	return 0, nil
}

// IsInterfaceNil -
func (w *WithBlockDataHandlerStub) IsInterfaceNil() bool {
	return w == nil
}
