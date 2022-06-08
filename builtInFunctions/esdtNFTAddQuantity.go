package builtInFunctions

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
)

const maxLenForAddNFTQuantity = 32

type esdtNFTAddQuantity struct {
	baseAlwaysActive
	keyPrefix             []byte
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	esdtStorageHandler    vmcommon.ESDTNFTStorageHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex

	valueLengthCheckEnableEpoch uint32
	flagValueLengthCheck        atomic.Flag
}

// NewESDTNFTAddQuantityFunc returns the esdt NFT add quantity built-in function component
func NewESDTNFTAddQuantityFunc(
	funcGasCost uint64,
	esdtStorageHandler vmcommon.ESDTNFTStorageHandler,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
	valueLengthCheckEnableEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*esdtNFTAddQuantity, error) {
	if check.IfNil(esdtStorageHandler) {
		return nil, ErrNilESDTNFTStorageHandler
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}

	e := &esdtNFTAddQuantity{
		keyPrefix:                   []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		globalSettingsHandler:       globalSettingsHandler,
		rolesHandler:                rolesHandler,
		funcGasCost:                 funcGasCost,
		mutExecution:                sync.RWMutex{},
		esdtStorageHandler:          esdtStorageHandler,
		valueLengthCheckEnableEpoch: valueLengthCheckEnableEpoch,
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *esdtNFTAddQuantity) EpochConfirmed(epoch uint32, _ uint64) {
	e.flagValueLengthCheck.SetValue(epoch >= e.valueLengthCheckEnableEpoch)
	log.Debug("ESDT Add Quantity value length check", "enabled", e.flagValueLengthCheck.IsSet())
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

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(core.ESDTRoleNFTAddQuantity))
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

	if e.flagValueLengthCheck.IsSet() && len(vmInput.Arguments[2]) > maxLenForAddNFTQuantity {
		return nil, fmt.Errorf("%w max length for add nft quantity is %d", ErrInvalidArguments, maxLenForAddNFTQuantity)
	}

	value := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	esdtData.Value.Add(esdtData.Value, value)

	_, err = e.esdtStorageHandler.SaveESDTNFTToken(acntSnd.AddressBytes(), acntSnd, esdtTokenKey, nonce, esdtData, false, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}
	err = e.esdtStorageHandler.AddToLiquiditySystemAcc(esdtTokenKey, nonce, value)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTNFTAddQuantity), vmInput.Arguments[0], nonce, value, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTAddQuantity) IsInterfaceNil() bool {
	return e == nil
}
