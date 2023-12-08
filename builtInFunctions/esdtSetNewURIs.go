package builtInFunctions

import (
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const uriStartIndex = 2

type esdtSetNewURIs struct {
	baseActiveHandler
	globalSettingsHandler vmcommon.GlobalMetadataHandler
	storageHandler        vmcommon.ESDTNFTStorageHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	accounts              vmcommon.AccountsAdapter
	enableEpochsHandler   vmcommon.EnableEpochsHandler
	blockDataHandler      vmcommon.BlockDataHandler
	funcGasCost           uint64
	gasConfig             vmcommon.BaseOperationCost
	mutExecution          sync.RWMutex
}

// NewESDTSetNewURIsFunc returns the esdt set new URIs built-in function component
func NewESDTSetNewURIsFunc(
	funcGasCost uint64,
	gasConfig vmcommon.BaseOperationCost,
	accounts vmcommon.AccountsAdapter,
	globalSettingsHandler vmcommon.GlobalMetadataHandler,
	storageHandler vmcommon.ESDTNFTStorageHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtSetNewURIs, error) {
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

	e := &esdtSetNewURIs{
		accounts:              accounts,
		globalSettingsHandler: globalSettingsHandler,
		storageHandler:        storageHandler,
		rolesHandler:          rolesHandler,
		funcGasCost:           funcGasCost,
		gasConfig:             gasConfig,
		mutExecution:          sync.RWMutex{},
		enableEpochsHandler:   enableEpochsHandler,
		blockDataHandler:      &disabledBlockDataHandler{},
	}

	e.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(DynamicEsdtFlag)
	}

	return e, nil
}

// ProcessBuiltinFunction saves the token type in the system account
func (e *esdtSetNewURIs) ProcessBuiltinFunction(acntSnd, _ vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	err := checkUpdateArguments(vmInput, acntSnd, e.baseActiveHandler, 3, e.rolesHandler, core.ESDTRoleSetNewURI)
	if err != nil {
		return nil, err
	}

	esdtInfo, err := getEsdtInfo(vmInput, acntSnd, e.storageHandler, e.globalSettingsHandler)
	if err != nil {
		return nil, err
	}

	oldURIsLen := lenArgs(esdtInfo.esdtData.TokenMetaData.URIs)
	newURIsLen := lenArgs(vmInput.Arguments[uriStartIndex:])
	difference := newURIsLen - oldURIsLen
	if difference < 0 {
		difference = 0
	}

	e.mutExecution.RLock()
	gasToUse := uint64(difference)*e.gasConfig.StorePerByte + e.funcGasCost
	e.mutExecution.RUnlock()

	if vmInput.GasProvided < gasToUse {
		return nil, ErrNotEnoughGas
	}

	esdtInfo.esdtData.TokenMetaData.URIs = vmInput.Arguments[uriStartIndex:]

	err = changeEsdtVersion(esdtInfo.esdtData, e.blockDataHandler.CurrentRound(), e.enableEpochsHandler)
	if err != nil {
		return nil, err
	}

	err = saveESDTMetaDataInfo(esdtInfo, e.storageHandler, acntSnd, vmInput.ReturnCallAfterError)
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
func (e *esdtSetNewURIs) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTSetNewURIs
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// SetBlockDataHandler is called when block data handler is set
func (e *esdtSetNewURIs) SetBlockDataHandler(blockDataHandler vmcommon.BlockDataHandler) error {
	if check.IfNil(blockDataHandler) {
		return ErrNilBlockDataHandler
	}

	e.blockDataHandler = blockDataHandler
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (e *esdtSetNewURIs) IsInterfaceNil() bool {
	return e == nil
}
