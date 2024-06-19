package builtInFunctions

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const (
	tokenIDIndex      = 0
	nonceIndex        = 1
	newRoyaltiesIndex = 2
)

type esdtModifyRoyalties struct {
	baseActiveHandler
	vmcommon.BlockchainDataProvider
	globalSettingsHandler vmcommon.GlobalMetadataHandler
	storageHandler        vmcommon.ESDTNFTStorageHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	accounts              vmcommon.AccountsAdapter
	enableEpochsHandler   vmcommon.EnableEpochsHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewESDTModifyRoyaltiesFunc returns the esdt modify royalties built-in function component
func NewESDTModifyRoyaltiesFunc(
	funcGasCost uint64,
	accounts vmcommon.AccountsAdapter,
	globalSettingsHandler vmcommon.GlobalMetadataHandler,
	storageHandler vmcommon.ESDTNFTStorageHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtModifyRoyalties, error) {
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

	e := &esdtModifyRoyalties{
		accounts:               accounts,
		globalSettingsHandler:  globalSettingsHandler,
		storageHandler:         storageHandler,
		rolesHandler:           rolesHandler,
		funcGasCost:            funcGasCost,
		mutExecution:           sync.RWMutex{},
		enableEpochsHandler:    enableEpochsHandler,
		BlockchainDataProvider: NewBlockchainDataProvider(),
	}

	e.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(DynamicEsdtFlag)
	}

	return e, nil
}

// ProcessBuiltinFunction saves the token type in the system account
func (e *esdtModifyRoyalties) ProcessBuiltinFunction(acntSnd, _ vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	err := checkUpdateArguments(vmInput, acntSnd, e.baseActiveHandler, 3, e.rolesHandler, core.ESDTRoleModifyRoyalties)
	if err != nil {
		return nil, err
	}

	e.mutExecution.RLock()
	funcGasCost := e.funcGasCost
	e.mutExecution.RUnlock()
	if vmInput.GasProvided < funcGasCost {
		return nil, ErrNotEnoughGas
	}

	esdtInfo, err := getEsdtInfo(vmInput, acntSnd, e.storageHandler, e.globalSettingsHandler)
	if err != nil {
		return nil, err
	}

	newRoyalties := uint32(big.NewInt(0).SetBytes(vmInput.Arguments[newRoyaltiesIndex]).Uint64())
	if newRoyalties > core.MaxRoyalty {
		return nil, fmt.Errorf("%w, invalid max royality value", ErrInvalidArguments)
	}

	esdtInfo.esdtData.TokenMetaData.Royalties = newRoyalties

	err = changeEsdtVersion(esdtInfo.esdtData, e.CurrentRound(), e.enableEpochsHandler)
	if err != nil {
		return nil, err
	}

	err = saveESDTMetaDataInfo(esdtInfo, e.storageHandler, acntSnd, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - funcGasCost,
	}

	extraTopics := [][]byte{vmInput.CallerAddr, vmInput.Arguments[newRoyaltiesIndex]}
	addESDTEntryInVMOutput(vmOutput, []byte(core.ESDTModifyRoyalties), vmInput.Arguments[tokenIDIndex], esdtInfo.esdtData.TokenMetaData.Nonce, big.NewInt(0), extraTopics...)

	return vmOutput, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtModifyRoyalties) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTModifyRoyalties
	e.mutExecution.Unlock()
}

// IsInterfaceNil returns true if there is no value under the interface
func (e *esdtModifyRoyalties) IsInterfaceNil() bool {
	return e == nil
}
