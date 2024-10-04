package builtInFunctions

import (
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/marshal"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const uriStartIndex = 2

type esdtSetNewURIs struct {
	baseActiveHandler
	vmcommon.BlockchainDataProvider
	globalSettingsHandler vmcommon.GlobalMetadataHandler
	storageHandler        vmcommon.ESDTNFTStorageHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	accounts              vmcommon.AccountsAdapter
	enableEpochsHandler   vmcommon.EnableEpochsHandler
	funcGasCost           uint64
	gasConfig             vmcommon.BaseOperationCost
	marshaller            marshal.Marshalizer
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
	marshaller marshal.Marshalizer,
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
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}

	e := &esdtSetNewURIs{
		accounts:               accounts,
		globalSettingsHandler:  globalSettingsHandler,
		storageHandler:         storageHandler,
		rolesHandler:           rolesHandler,
		funcGasCost:            funcGasCost,
		gasConfig:              gasConfig,
		mutExecution:           sync.RWMutex{},
		enableEpochsHandler:    enableEpochsHandler,
		BlockchainDataProvider: NewBlockchainDataProvider(),
		marshaller:             marshaller,
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

	metaDataVersion, _, err := getMetaDataVersion(esdtInfo.esdtData, e.enableEpochsHandler, e.marshaller)
	if err != nil {
		return nil, err
	}

	esdtInfo.esdtData.TokenMetaData.URIs = vmInput.Arguments[uriStartIndex:]
	metaDataVersion.URIs = e.CurrentRound()

	err = changeEsdtVersion(esdtInfo.esdtData, metaDataVersion, e.enableEpochsHandler, e.marshaller)
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

	extraTopics := append([][]byte{vmInput.CallerAddr}, vmInput.Arguments[uriStartIndex:]...)
	addESDTEntryInVMOutput(vmOutput, []byte(core.ESDTSetNewURIs), vmInput.Arguments[0], esdtInfo.esdtData.TokenMetaData.Nonce, big.NewInt(0), extraTopics...)

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

// IsInterfaceNil returns true if there is no value under the interface
func (e *esdtSetNewURIs) IsInterfaceNil() bool {
	return e == nil
}
