package builtInFunctions

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type esdtMetaDataUpdate struct {
	baseActiveHandler
	funcGasCost           uint64
	globalSettingsHandler vmcommon.GlobalMetadataHandler
	storageHandler        vmcommon.ESDTNFTStorageHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	accounts              vmcommon.AccountsAdapter
	enableEpochsHandler   vmcommon.EnableEpochsHandler
	gasConfig             vmcommon.BaseOperationCost
	mutExecution          sync.RWMutex
}

// NewESDTMetaDataUpdateFunc returns the esdt meta data update built-in function component
func NewESDTMetaDataUpdateFunc(
	funcGasCost uint64,
	gasConfig vmcommon.BaseOperationCost,
	accounts vmcommon.AccountsAdapter,
	globalSettingsHandler vmcommon.GlobalMetadataHandler,
	storageHandler vmcommon.ESDTNFTStorageHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtMetaDataUpdate, error) {
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if check.IfNil(storageHandler) {
		return nil, ErrNilESDTNFTStorageHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}

	e := &esdtMetaDataUpdate{
		accounts:              accounts,
		globalSettingsHandler: globalSettingsHandler,
		storageHandler:        storageHandler,
		rolesHandler:          rolesHandler,
		enableEpochsHandler:   enableEpochsHandler,
		funcGasCost:           funcGasCost,
		gasConfig:             gasConfig,
		mutExecution:          sync.RWMutex{},
	}

	e.baseActiveHandler.activeHandler = enableEpochsHandler.IsDynamicESDTEnabled

	return e, nil
}

// ProcessBuiltinFunction saves the token type in the system account
func (e *esdtMetaDataUpdate) ProcessBuiltinFunction(acntSnd, _ vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	err := checkArguments(vmInput, acntSnd, e.baseActiveHandler)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 7 {
		return nil, ErrInvalidNumberOfArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[tokenIDIndex], []byte(core.ESDTRoleNFTUpdate))
	if err != nil {
		return nil, err
	}

	totalLength := uint64(0)
	for _, arg := range vmInput.Arguments {
		totalLength += uint64(len(arg))
	}

	e.mutExecution.RLock()
	gasToUse := totalLength*e.gasConfig.StorePerByte + e.funcGasCost
	e.mutExecution.RUnlock()
	if vmInput.GasProvided < gasToUse {
		return nil, ErrNotEnoughGas
	}

	esdtData, esdtTokenKey, nonce, err := getEsdtDataAndCheckType(vmInput, acntSnd, e.storageHandler)
	if err != nil {
		return nil, err
	}

	if len(vmInput.Arguments[nameIndex]) != 0 {
		esdtData.TokenMetaData.Name = vmInput.Arguments[nameIndex]
	}
	esdtData.TokenMetaData.Creator = vmInput.CallerAddr

	if len(vmInput.Arguments[royaltiesIndex]) != 0 {
		royalties := uint32(big.NewInt(0).SetBytes(vmInput.Arguments[royaltiesIndex]).Uint64())
		if royalties > core.MaxRoyalty {
			return nil, fmt.Errorf("%w, invalid max royality value", ErrInvalidArguments)
		}
		esdtData.TokenMetaData.Royalties = royalties
	}

	if len(vmInput.Arguments[hashIndex]) != 0 {
		esdtData.TokenMetaData.Hash = vmInput.Arguments[hashIndex]
	}

	if len(vmInput.Arguments[attributesIndex]) != 0 {
		esdtData.TokenMetaData.Attributes = vmInput.Arguments[attributesIndex]
	}

	var URIs [][]byte
	for _, arg := range vmInput.Arguments[urisStartIndex:] {
		if len(arg) != 0 {
			URIs = append(URIs, arg)
		}
	}

	if len(URIs) != 0 {
		esdtData.TokenMetaData.URIs = URIs
	}

	_, err = e.storageHandler.SaveESDTNFTToken(acntSnd.AddressBytes(), acntSnd, esdtTokenKey, nonce, esdtData, true, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - gasToUse,
	}
	return vmOutput, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtMetaDataUpdate) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTUpdate
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// IsInterfaceNil returns true if there is no value under the interface
func (e *esdtMetaDataUpdate) IsInterfaceNil() bool {
	return e == nil
}
