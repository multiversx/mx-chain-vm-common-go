package builtInFunctions

import (
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
)

type esdtNFTBurn struct {
	baseAlwaysActive
	keyPrefix             []byte
	esdtStorageHandler    vmcommon.ESDTNFTStorageHandler
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewESDTNFTBurnFunc returns the esdt NFT burn built-in function component
func NewESDTNFTBurnFunc(
	funcGasCost uint64,
	esdtStorageHandler vmcommon.ESDTNFTStorageHandler,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
) (*esdtNFTBurn, error) {
	if check.IfNil(esdtStorageHandler) {
		return nil, ErrNilESDTNFTStorageHandler
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}

	e := &esdtNFTBurn{
		keyPrefix:             []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		esdtStorageHandler:    esdtStorageHandler,
		globalSettingsHandler: globalSettingsHandler,
		rolesHandler:          rolesHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTBurn) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTBurn
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT NFT burn function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - quantity to burn
func (e *esdtNFTBurn) ProcessBuiltinFunction(
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

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(core.ESDTRoleNFTBurn))
	if err != nil {
		return nil, err
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	esdtData, err := e.esdtStorageHandler.GetESDTNFTTokenOnSender(acntSnd, esdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}

	quantityToBurn := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	if esdtData.Value.Cmp(quantityToBurn) < 0 {
		return nil, ErrInvalidNFTQuantity
	}

	esdtData.Value.Sub(esdtData.Value, quantityToBurn)

	_, err = e.esdtStorageHandler.SaveESDTNFTToken(acntSnd.AddressBytes(), acntSnd, esdtTokenKey, nonce, esdtData, false, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTNFTBurn), vmInput.Arguments[0], nonce, quantityToBurn, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTBurn) IsInterfaceNil() bool {
	return e == nil
}
