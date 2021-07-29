package builtInFunctions

import (
	"bytes"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/atomic"
)

type esdtBurn struct {
	*baseDisabled
	funcGasCost           uint64
	marshalizer           vmcommon.Marshalizer
	keyPrefix             []byte
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler
	mutExecution          sync.RWMutex
}

// NewESDTBurnFunc returns the esdt burn built-in function component
func NewESDTBurnFunc(
	funcGasCost uint64,
	marshalizer vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	disableEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*esdtBurn, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}

	e := &esdtBurn{
		funcGasCost:           funcGasCost,
		marshalizer:           marshalizer,
		keyPrefix:             []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		globalSettingsHandler: globalSettingsHandler,
	}

	e.baseDisabled = &baseDisabled{
		function:          core.BuiltInFunctionESDTBurn,
		deActivationEpoch: disableEpoch,
		flagActivated:     atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtBurn) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTBurn
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT burn function call
func (e *esdtBurn) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) != 2 {
		return nil, ErrInvalidArguments
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return nil, ErrNegativeValue
	}
	if !bytes.Equal(vmInput.RecipientAddr, core.ESDTSCAddress) {
		return nil, ErrAddressIsNotESDTSystemSC
	}
	if check.IfNil(acntSnd) {
		return nil, ErrNilUserAccount
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)

	if vmInput.GasProvided < e.funcGasCost {
		return nil, ErrNotEnoughGas
	}

	err = addToESDTBalance(acntSnd, esdtTokenKey, big.NewInt(0).Neg(value), e.marshalizer, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	gasRemaining := computeGasRemaining(acntSnd, vmInput.GasProvided, e.funcGasCost)
	vmOutput := &vmcommon.VMOutput{GasRemaining: gasRemaining, ReturnCode: vmcommon.Ok}
	if vmcommon.IsSmartContractAddress(vmInput.CallerAddr) {
		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			core.BuiltInFunctionESDTBurn,
			vmInput.Arguments,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTBurn), vmInput.Arguments[0], value, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtBurn) IsInterfaceNil() bool {
	return e == nil
}
