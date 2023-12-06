package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const (
	nameIndex       = 2
	royaltiesIndex  = 3
	hashIndex       = 4
	attributesIndex = 5
	urisStartIndex  = 6
)

type esdtMetaDataRecreate struct {
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

// NewESDTMetaDataRecreateFunc returns the esdt meta data recreate built-in function component
func NewESDTMetaDataRecreateFunc(
	funcGasCost uint64,
	gasConfig vmcommon.BaseOperationCost,
	accounts vmcommon.AccountsAdapter,
	globalSettingsHandler vmcommon.GlobalMetadataHandler,
	storageHandler vmcommon.ESDTNFTStorageHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtMetaDataRecreate, error) {
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

	e := &esdtMetaDataRecreate{
		accounts:              accounts,
		globalSettingsHandler: globalSettingsHandler,
		storageHandler:        storageHandler,
		rolesHandler:          rolesHandler,
		enableEpochsHandler:   enableEpochsHandler,
		funcGasCost:           funcGasCost,
		gasConfig:             gasConfig,
		mutExecution:          sync.RWMutex{},
	}

	e.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(DynamicEsdtFlag)
	}

	return e, nil
}

func checkUpdateArguments(
	vmInput *vmcommon.ContractCallInput,
	acntSnd vmcommon.UserAccountHandler,
	handler baseActiveHandler,
	minNumOfArgs int,
	rolesHandler vmcommon.ESDTRoleHandler,
	role string,
) error {
	if vmInput == nil {
		return ErrNilVmInput
	}
	if vmInput.CallValue == nil {
		return ErrNilValue
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return ErrBuiltInFunctionCalledWithValue
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return ErrInvalidRcvAddr
	}
	if check.IfNil(acntSnd) {
		return ErrNilUserAccount
	}
	if !handler.IsActive() {
		return ErrBuiltInFunctionIsNotActive
	}
	if len(vmInput.Arguments) < minNumOfArgs {
		return ErrInvalidNumberOfArguments
	}
	if minNumOfArgs < 1 {
		return ErrInvalidNumberOfArguments
	}

	tokenId := append([]byte(baseESDTKeyPrefix), vmInput.Arguments[tokenIDIndex]...)
	return rolesHandler.CheckAllowedToExecute(acntSnd, tokenId, []byte(role))
}

type esdtStorageInfo struct {
	esdtData     *esdt.ESDigitalToken
	esdtTokenKey []byte
	nonce        uint64
	isDynamic    bool
}

func getEsdtInfo(
	vmInput *vmcommon.ContractCallInput,
	acntSnd vmcommon.UserAccountHandler,
	storageHandler vmcommon.ESDTNFTStorageHandler,
	globalSettingsHandler vmcommon.GlobalMetadataHandler,
) (*esdtStorageInfo, error) {
	esdtTokenKey := append([]byte(baseESDTKeyPrefix), vmInput.Arguments[tokenIDIndex]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[nonceIndex]).Uint64()

	tokenType, err := globalSettingsHandler.GetTokenType(esdtTokenKey)
	if err != nil {
		return nil, err
	}
	if core.IsDynamicESDT(tokenType) {
		metaData, err := storageHandler.GetMetaDataFromSystemAccount(esdtTokenKey, nonce)
		if err != nil {
			return nil, err
		}

		if metaData == nil {
			return nil, ErrInvalidMetadata
		}

		esdtData := &esdt.ESDigitalToken{TokenMetaData: metaData}
		return &esdtStorageInfo{
			esdtData:     esdtData,
			esdtTokenKey: esdtTokenKey,
			nonce:        nonce,
			isDynamic:    true,
		}, nil
	}

	esdtData, err := storageHandler.GetESDTNFTTokenOnSender(acntSnd, esdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}

	if esdtData.Value == nil || esdtData.Value.Cmp(zero) == 0 {
		return nil, ErrInvalidEsdtValue
	}

	return &esdtStorageInfo{
		esdtData:     esdtData,
		esdtTokenKey: esdtTokenKey,
		nonce:        nonce,
		isDynamic:    false,
	}, nil
}

func saveESDTMetaDataInfo(
	esdtInfo *esdtStorageInfo,
	storageHandler vmcommon.ESDTNFTStorageHandler,
	acntSnd vmcommon.UserAccountHandler,
	returnCallAfterError bool,
) error {
	if esdtInfo.isDynamic {
		return storageHandler.SaveMetaDataToSystemAccount(esdtInfo.esdtTokenKey, esdtInfo.nonce, esdtInfo.esdtData)
	}

	if esdtInfo.esdtData.Value == nil || esdtInfo.esdtData.Value.Cmp(zero) == 0 {
		return ErrInvalidEsdtValue
	}

	_, err := storageHandler.SaveESDTNFTToken(acntSnd.AddressBytes(), acntSnd, esdtInfo.esdtTokenKey, esdtInfo.nonce, esdtInfo.esdtData, true, returnCallAfterError)
	return err
}

func lenArgs(args [][]byte) int {
	totalLength := 0
	for _, arg := range args {
		totalLength += len(arg)
	}
	return totalLength
}

// ProcessBuiltinFunction saves the token type in the system account
func (e *esdtMetaDataRecreate) ProcessBuiltinFunction(acntSnd, _ vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	err := checkUpdateArguments(vmInput, acntSnd, e.baseActiveHandler, 7, e.rolesHandler, core.ESDTMetaDataRecreate)
	if err != nil {
		return nil, err
	}

	totalLengthDifference := lenArgs(vmInput.Arguments)

	esdtInfo, err := getEsdtInfo(vmInput, acntSnd, e.storageHandler, e.globalSettingsHandler)
	if err != nil {
		return nil, err
	}

	totalLengthDifference -= esdtInfo.esdtData.TokenMetaData.Size()
	if totalLengthDifference < 0 {
		totalLengthDifference = 0
	}

	e.mutExecution.RLock()
	gasToUse := uint64(totalLengthDifference)*e.gasConfig.StorePerByte + e.funcGasCost
	e.mutExecution.RUnlock()
	if vmInput.GasProvided < gasToUse {
		return nil, ErrNotEnoughGas
	}

	royalties := uint32(big.NewInt(0).SetBytes(vmInput.Arguments[royaltiesIndex]).Uint64())
	if royalties > core.MaxRoyalty {
		return nil, fmt.Errorf("%w, invalid max royality value", ErrInvalidArguments)
	}

	esdtInfo.esdtData.TokenMetaData.Name = vmInput.Arguments[nameIndex]
	esdtInfo.esdtData.TokenMetaData.Creator = vmInput.CallerAddr
	esdtInfo.esdtData.TokenMetaData.Royalties = royalties
	esdtInfo.esdtData.TokenMetaData.Hash = vmInput.Arguments[hashIndex]
	esdtInfo.esdtData.TokenMetaData.Attributes = vmInput.Arguments[attributesIndex]
	esdtInfo.esdtData.TokenMetaData.URIs = vmInput.Arguments[urisStartIndex:]

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
func (e *esdtMetaDataRecreate) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTRecreate
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// IsInterfaceNil returns true if there is no value under the interface
func (e *esdtMetaDataRecreate) IsInterfaceNil() bool {
	return e == nil
}
