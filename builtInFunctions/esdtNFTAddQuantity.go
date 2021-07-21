package builtInFunctions

import (
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
)

type esdtNFTAddQuantity struct {
	baseAlwaysActive
	keyPrefix    []byte
	marshalizer  vmcommon.Marshalizer
	pauseHandler vmcommon.ESDTPauseHandler
	rolesHandler vmcommon.ESDTRoleHandler
	funcGasCost  uint64
	mutExecution sync.RWMutex
}

// NewESDTNFTAddQuantityFunc returns the esdt NFT add quantity built-in function component
func NewESDTNFTAddQuantityFunc(
	funcGasCost uint64,
	marshalizer vmcommon.Marshalizer,
	pauseHandler vmcommon.ESDTPauseHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
) (*esdtNFTAddQuantity, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(pauseHandler) {
		return nil, ErrNilPauseHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}

	e := &esdtNFTAddQuantity{
		keyPrefix:    []byte(vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTKeyIdentifier),
		marshalizer:  marshalizer,
		pauseHandler: pauseHandler,
		rolesHandler: rolesHandler,
		funcGasCost:  funcGasCost,
		mutExecution: sync.RWMutex{},
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTAddQuantity) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTAddQuantity
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT NFT add quantity function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - quantity to add
func (e *esdtNFTAddQuantity) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkESDTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 3 {
		return nil, ErrInvalidArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(vmcommon.ESDTRoleNFTAddQuantity))
	if err != nil {
		return nil, err
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}
	esdtData, err := getESDTNFTTokenOnSender(acntSnd, esdtTokenKey, nonce, e.marshalizer)
	if err != nil {
		return nil, err
	}

	esdtData.Value.Add(esdtData.Value, big.NewInt(0).SetBytes(vmInput.Arguments[2]))

	_, err = saveESDTNFTToken(acntSnd, esdtTokenKey, esdtData, e.marshalizer, e.pauseHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	logEntry := newEntryForNFT(vmcommon.BuiltInFunctionESDTNFTAddQuantity, vmInput.CallerAddr, vmInput.Arguments[0], nonce)
	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
		Logs:         []*vmcommon.LogEntry{logEntry},
	}
	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTAddQuantity) IsInterfaceNil() bool {
	return e == nil
}
