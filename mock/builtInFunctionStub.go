package mock

import (
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

// BuiltInFunctionStub -
type BuiltInFunctionStub struct {
	ProcessBuiltinFunctionCalled func(acntSnd, acntDst vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error)
	SetNewGasConfigCalled        func(gasCost *vmcommon.GasCost)
	IsActiveCalled               func() bool
	SetBlockchainHookCalled      func(blockchainHook vmcommon.BlockchainDataHook) error
	CurrentRoundCalled           func() uint64
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

// SetBlockchainHook -
func (b *BuiltInFunctionStub) SetBlockchainHook(blockchainHook vmcommon.BlockchainDataHook) error {
	if b.SetBlockchainHookCalled != nil {
		return b.SetBlockchainHookCalled(blockchainHook)
	}
	return nil
}

// CurrentRound -
func (b *BuiltInFunctionStub) CurrentRound() uint64 {
	if b.CurrentRoundCalled != nil {
		return b.CurrentRoundCalled()
	}
	return 0
}

// IsInterfaceNil -
func (b *BuiltInFunctionStub) IsInterfaceNil() bool {
	return b == nil
}
