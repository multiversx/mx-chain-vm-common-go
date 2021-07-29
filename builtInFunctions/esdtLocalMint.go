package builtInFunctions

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
)

type esdtLocalMint struct {
	baseAlwaysActive
	keyPrefix             []byte
	marshalizer           vmcommon.Marshalizer
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewESDTLocalMintFunc returns the esdt local mint built-in function component
func NewESDTLocalMintFunc(
	funcGasCost uint64,
	marshalizer vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
) (*esdtLocalMint, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}

	e := &esdtLocalMint{
		keyPrefix:             []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		marshalizer:           marshalizer,
		globalSettingsHandler: globalSettingsHandler,
		rolesHandler:          rolesHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtLocalMint) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTLocalMint
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT local mint function call
func (e *esdtLocalMint) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkInputArgumentsForLocalAction(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}

	tokenID := vmInput.Arguments[0]
	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(core.ESDTRoleLocalMint))
	if err != nil {
		return nil, err
	}

	if len(vmInput.Arguments[1]) > core.MaxLenForESDTIssueMint {
		return nil, fmt.Errorf("%w max length for esdt issue is %d", ErrInvalidArguments, core.MaxLenForESDTIssueMint)
	}

	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	esdtTokenKey := append(e.keyPrefix, tokenID...)
	err = addToESDTBalance(acntSnd, esdtTokenKey, big.NewInt(0).Set(value), e.marshalizer, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - e.funcGasCost}

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTLocalMint), vmInput.Arguments[0], value, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtLocalMint) IsInterfaceNil() bool {
	return e == nil
}
