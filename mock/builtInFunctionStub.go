package mock

import (
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

// BuiltInFunctionStub -
type BuiltInFunctionStub struct {
	ProcessBuiltinFunctionCalled func(acntSnd, acntDst vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error)
	SetNewGasConfigCalled        func(gasCost *vmcommon.GasCost)
	IsActiveCalled               func() bool
	SetBlockDataHandlerCalled    func(blockDataHandler vmcommon.BlockDataHandler) error
	CurrentRoundCalled           func() (uint64, error)
}

// ProcessBuiltinFunction -
func (b *BuiltInFunctionStub) ProcessBuiltinFunction(acntSnd, acntDst vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	if b.ProcessBuiltinFunctionCalled != nil {
		return b.ProcessBuiltinFunctionCalled(acntSnd, acntDst, vmInput)
	}
	return &vmcommon.VMOutput{}, nil
}

// SetNewGasConfig -
func (b *BuiltInFunctionStub) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if b.SetNewGasConfigCalled != nil {
		b.SetNewGasConfigCalled(gasCost)
	}
}

// IsActive -
func (b *BuiltInFunctionStub) IsActive() bool {
	if b.IsActiveCalled != nil {
		return b.IsActiveCalled()
	}
	return true
}

// SetBlockDataHandler -
func (b *BuiltInFunctionStub) SetBlockDataHandler(blockDataHandler vmcommon.BlockDataHandler) error {
	if b.SetBlockDataHandlerCalled != nil {
		return b.SetBlockDataHandlerCalled(blockDataHandler)
	}
	return nil
}

// CurrentRound -
func (b *BuiltInFunctionStub) CurrentRound() (uint64, error) {
	if b.CurrentRoundCalled != nil {
		return b.CurrentRoundCalled()
	}
	return 0, nil
}

// IsInterfaceNil -
func (b *BuiltInFunctionStub) IsInterfaceNil() bool {
	return b == nil
}
