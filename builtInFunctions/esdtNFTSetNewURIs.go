package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const uriStartIndex = 2

type esdtNFTSetNewURIs struct {
	baseActiveHandler
	globalSettingsHandler vmcommon.GlobalMetadataHandler
	storageHandler        vmcommon.ESDTNFTStorageHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	accounts              vmcommon.AccountsAdapter
}

// NewESDTNFTSetNewURIsFunc returns the esdt set new URIs built-in function component
func NewESDTNFTSetNewURIsFunc(
	accounts vmcommon.AccountsAdapter,
	globalSettingsHandler vmcommon.GlobalMetadataHandler,
	storageHandler vmcommon.ESDTNFTStorageHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtNFTSetNewURIs, error) {
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

	e := &esdtNFTSetNewURIs{
		accounts:              accounts,
		globalSettingsHandler: globalSettingsHandler,
		storageHandler:        storageHandler,
		rolesHandler:          rolesHandler,
	}

	e.baseActiveHandler.activeHandler = enableEpochsHandler.IsDynamicESDTEnabled

	return e, nil
}

// ProcessBuiltinFunction saves the token type in the system account
func (e *esdtNFTSetNewURIs) ProcessBuiltinFunction(acntSnd, _ vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue == nil {
		return nil, ErrNilValue
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) < 3 {
		return nil, ErrInvalidNumberOfArguments
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return nil, ErrInvalidRcvAddr
	}
	if check.IfNil(acntSnd) {
		return nil, ErrNilUserAccount
	}
	if !e.baseActiveHandler.IsActive() {
		return nil, ErrBuiltInFunctionIsNotActive
	}
	// TODO check and consume gas

	err := e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[tokenIDIndex], []byte(core.ESDTRoleSetNewURI))
	if err != nil {
		return nil, err
	}

	esdtTokenKey := append([]byte(baseESDTKeyPrefix), vmInput.Arguments[tokenIDIndex]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[nonceIndex]).Uint64()
	esdtData, err := e.storageHandler.GetESDTNFTTokenOnSender(acntSnd, esdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if esdtData.Type != uint32(core.DynamicNFT) {
		return nil, ErrOperationNotPermitted
	}

	esdtData.TokenMetaData.URIs = vmInput.Arguments[uriStartIndex:]

	_, err = e.storageHandler.SaveESDTNFTToken(acntSnd.AddressBytes(), acntSnd, esdtTokenKey, nonce, esdtData, true, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode: vmcommon.Ok,
		//TODO set GasRemaining
	}
	return vmOutput, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTSetNewURIs) SetNewGasConfig(_ *vmcommon.GasCost) {
	//TODO set gas cost
}

// IsInterfaceNil returns true if there is no value under the interface
func (e *esdtNFTSetNewURIs) IsInterfaceNil() bool {
	return e == nil
}
