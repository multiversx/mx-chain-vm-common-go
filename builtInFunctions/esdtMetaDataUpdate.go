package builtInFunctions

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/marshal"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type esdtMetaDataUpdate struct {
	baseActiveHandler
	vmcommon.BlockchainDataProvider
	funcGasCost           uint64
	globalSettingsHandler vmcommon.GlobalMetadataHandler
	storageHandler        vmcommon.ESDTNFTStorageHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	accounts              vmcommon.AccountsAdapter
	enableEpochsHandler   vmcommon.EnableEpochsHandler
	gasConfig             vmcommon.BaseOperationCost
	marshaller            marshal.Marshalizer
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
	marshaller marshal.Marshalizer,
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
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}

	e := &esdtMetaDataUpdate{
		accounts:               accounts,
		globalSettingsHandler:  globalSettingsHandler,
		storageHandler:         storageHandler,
		rolesHandler:           rolesHandler,
		enableEpochsHandler:    enableEpochsHandler,
		funcGasCost:            funcGasCost,
		gasConfig:              gasConfig,
		mutExecution:           sync.RWMutex{},
		BlockchainDataProvider: NewBlockchainDataProvider(),
		marshaller:             marshaller,
	}

	e.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(DynamicEsdtFlag)
	}

	return e, nil
}

// ProcessBuiltinFunction saves the token type in the system account
func (e *esdtMetaDataUpdate) ProcessBuiltinFunction(acntSnd, _ vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	err := checkUpdateArguments(vmInput, acntSnd, e.baseActiveHandler, 7, e.rolesHandler, core.ESDTRoleNFTUpdate)
	if err != nil {
		return nil, err
	}

	totalLengthDifference := lenArgs(vmInput.Arguments)

	esdtInfo, err := getEsdtInfo(vmInput, acntSnd, e.storageHandler, e.globalSettingsHandler)
	if err != nil {
		return nil, err
	}

	currentRound := e.CurrentRound()
	metaDataVersion, _, err := getMetaDataVersion(esdtInfo.esdtData, e.enableEpochsHandler, e.marshaller)
	if err != nil {
		return nil, err
	}

	if len(vmInput.Arguments[nameIndex]) != 0 {
		totalLengthDifference -= len(esdtInfo.esdtData.TokenMetaData.Name)
		esdtInfo.esdtData.TokenMetaData.Name = vmInput.Arguments[nameIndex]
		metaDataVersion.Name = currentRound
	}
	totalLengthDifference -= len(esdtInfo.esdtData.TokenMetaData.Creator)
	esdtInfo.esdtData.TokenMetaData.Creator = vmInput.CallerAddr
	metaDataVersion.Creator = currentRound

	if len(vmInput.Arguments[royaltiesIndex]) != 0 {
		totalLengthDifference -= len(vmInput.Arguments[royaltiesIndex])
		royalties := uint32(big.NewInt(0).SetBytes(vmInput.Arguments[royaltiesIndex]).Uint64())
		if royalties > core.MaxRoyalty {
			return nil, fmt.Errorf("%w, invalid max royality value", ErrInvalidArguments)
		}
		esdtInfo.esdtData.TokenMetaData.Royalties = royalties
		metaDataVersion.Royalties = currentRound
	}

	if len(vmInput.Arguments[hashIndex]) != 0 {
		totalLengthDifference -= len(esdtInfo.esdtData.TokenMetaData.Hash)
		esdtInfo.esdtData.TokenMetaData.Hash = vmInput.Arguments[hashIndex]
		metaDataVersion.Hash = currentRound
	}

	if len(vmInput.Arguments[attributesIndex]) != 0 {
		totalLengthDifference -= len(esdtInfo.esdtData.TokenMetaData.Attributes)
		esdtInfo.esdtData.TokenMetaData.Attributes = vmInput.Arguments[attributesIndex]
		metaDataVersion.Attributes = currentRound
	}

	if len(vmInput.Arguments[urisStartIndex]) != 0 {
		for _, uri := range esdtInfo.esdtData.TokenMetaData.URIs {
			totalLengthDifference -= len(uri)
		}

		esdtInfo.esdtData.TokenMetaData.URIs = vmInput.Arguments[urisStartIndex:]
		metaDataVersion.URIs = currentRound
	}

	if totalLengthDifference < 0 {
		totalLengthDifference = 0
	}

	e.mutExecution.RLock()
	gasToUse := uint64(totalLengthDifference)*e.gasConfig.StorePerByte + e.funcGasCost
	e.mutExecution.RUnlock()
	if vmInput.GasProvided < gasToUse {
		return nil, ErrNotEnoughGas
	}

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

	esdtDataBytes, err := e.marshaller.Marshal(esdtInfo.esdtData)
	if err != nil {
		log.Warn("esdtMetaDataUpdate.ProcessBuiltinFunction: cannot marshall esdt data for log", "error", err)
	}

	addESDTEntryInVMOutput(vmOutput, []byte(core.ESDTMetaDataUpdate), vmInput.Arguments[0], esdtInfo.esdtData.TokenMetaData.Nonce, big.NewInt(0), vmInput.CallerAddr, esdtDataBytes)

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
