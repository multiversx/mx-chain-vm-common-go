package builtInFunctions

import (
	"bytes"
	"fmt"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
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

	e.baseActiveHandler.activeHandler = enableEpochsHandler.IsDynamicESDTEnabled

	return e, nil
}

func checkArguments(vmInput *vmcommon.ContractCallInput, acntSnd vmcommon.UserAccountHandler, handler baseActiveHandler) error {
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

	return nil
}

func getEsdtDataAndCheckType(
	vmInput *vmcommon.ContractCallInput,
	acntSnd vmcommon.UserAccountHandler,
	storageHandler vmcommon.ESDTNFTStorageHandler,
) (*esdt.ESDigitalToken, []byte, uint64, error) {
	esdtTokenKey := append([]byte(baseESDTKeyPrefix), vmInput.Arguments[tokenIDIndex]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[nonceIndex]).Uint64()
	esdtData, err := storageHandler.GetESDTNFTTokenOnSender(acntSnd, esdtTokenKey, nonce)
	if err != nil {
		return nil, nil, 0, err
	}
	if !core.IsDynamicESDT(esdtData.Type) {
		return nil, nil, 0, ErrOperationNotPermitted
	}

	return esdtData, esdtTokenKey, nonce, nil
}

// ProcessBuiltinFunction saves the token type in the system account
func (e *esdtMetaDataRecreate) ProcessBuiltinFunction(acntSnd, _ vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	err := checkArguments(vmInput, acntSnd, e.baseActiveHandler)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 7 {
		return nil, ErrInvalidNumberOfArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[tokenIDIndex], []byte(core.ESDTRoleModifyCreator))
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

	royalties := uint32(big.NewInt(0).SetBytes(vmInput.Arguments[royaltiesIndex]).Uint64())
	if royalties > core.MaxRoyalty {
		return nil, fmt.Errorf("%w, invalid max royality value", ErrInvalidArguments)
	}

	esdtData.TokenMetaData.Name = vmInput.Arguments[nameIndex]
	esdtData.TokenMetaData.Creator = vmInput.CallerAddr
	esdtData.TokenMetaData.Royalties = royalties
	esdtData.TokenMetaData.Hash = vmInput.Arguments[hashIndex]
	esdtData.TokenMetaData.Attributes = vmInput.Arguments[attributesIndex]
	esdtData.TokenMetaData.URIs = vmInput.Arguments[urisStartIndex:]

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
