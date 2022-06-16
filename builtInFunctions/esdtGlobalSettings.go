package builtInFunctions

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
)

type esdtGlobalSettings struct {
	*baseEnabled
	keyPrefix           []byte
	set                 bool
	accounts            vmcommon.AccountsAdapter
	enableEpochsHandler vmcommon.EnableEpochsHandler
}

// NewESDTGlobalSettingsFunc returns the esdt pause/un-pause built-in function component
func NewESDTGlobalSettingsFunc(
	accounts vmcommon.AccountsAdapter,
	set bool,
	function string,
	activationFlagName string,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtGlobalSettings, error) {
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if !isCorrectFunction(function) {
		return nil, ErrInvalidArguments
	}

	e := &esdtGlobalSettings{
		keyPrefix: []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		set:       set,
		accounts:  accounts,
	}

	e.baseEnabled = &baseEnabled{
		function:            function,
		activationFlagName:  activationFlagName,
		enableEpochsHandler: enableEpochsHandler,
	}

	return e, nil
}

func isCorrectFunction(function string) bool {
	switch function {
	case core.BuiltInFunctionESDTPause, core.BuiltInFunctionESDTUnPause, core.BuiltInFunctionESDTSetLimitedTransfer, core.BuiltInFunctionESDTUnSetLimitedTransfer:
		return true
	case vmcommon.BuiltInFunctionESDTSetBurnRoleForAll, vmcommon.BuiltInFunctionESDTUnSetBurnRoleForAll:
		return true
	default:
		return false
	}
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtGlobalSettings) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves ESDT pause function call
func (e *esdtGlobalSettings) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return nil, ErrInvalidArguments
	}
	if !bytes.Equal(vmInput.CallerAddr, core.ESDTSCAddress) {
		return nil, ErrAddressIsNotESDTSystemSC
	}
	if !vmcommon.IsSystemAccountAddress(vmInput.RecipientAddr) {
		return nil, ErrOnlySystemAccountAccepted
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)

	err := e.toggleSetting(esdtTokenKey)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	return vmOutput, nil
}

func (e *esdtGlobalSettings) toggleSetting(esdtTokenKey []byte) error {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	esdtMetaData, err := e.getGlobalMetadata(esdtTokenKey)
	if err != nil {
		return err
	}

	switch e.function {
	case core.BuiltInFunctionESDTSetLimitedTransfer, core.BuiltInFunctionESDTUnSetLimitedTransfer:
		esdtMetaData.LimitedTransfer = e.set
		break
	case core.BuiltInFunctionESDTPause, core.BuiltInFunctionESDTUnPause:
		esdtMetaData.Paused = e.set
		break
	case vmcommon.BuiltInFunctionESDTUnSetBurnRoleForAll, vmcommon.BuiltInFunctionESDTSetBurnRoleForAll:
		esdtMetaData.BurnRoleForAll = e.set
		break
	}

	err = systemSCAccount.AccountDataHandler().SaveKeyValue(esdtTokenKey, esdtMetaData.ToBytes())
	if err != nil {
		return err
	}

	return e.accounts.SaveAccount(systemSCAccount)
}

func (e *esdtGlobalSettings) getSystemAccount() (vmcommon.UserAccountHandler, error) {
	systemSCAccount, err := e.accounts.LoadAccount(vmcommon.SystemAccountAddress)
	if err != nil {
		return nil, err
	}

	userAcc, ok := systemSCAccount.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAcc, nil
}

// IsPaused returns true if the esdtTokenKey (prefixed) is paused
func (e *esdtGlobalSettings) IsPaused(esdtTokenKey []byte) bool {
	esdtMetadata, err := e.getGlobalMetadata(esdtTokenKey)
	if err != nil {
		return false
	}

	return esdtMetadata.Paused
}

// IsLimitedTransfer returns true if the esdtTokenKey (prefixed) is with limited transfer
func (e *esdtGlobalSettings) IsLimitedTransfer(esdtTokenKey []byte) bool {
	esdtMetadata, err := e.getGlobalMetadata(esdtTokenKey)
	if err != nil {
		return false
	}

	return esdtMetadata.LimitedTransfer
}

// IsBurnForAll returns true if the esdtTokenKey (prefixed) is with burn for all
func (e *esdtGlobalSettings) IsBurnForAll(esdtTokenKey []byte) bool {
	esdtMetadata, err := e.getGlobalMetadata(esdtTokenKey)
	if err != nil {
		return false
	}

	return esdtMetadata.BurnRoleForAll
}

func (e *esdtGlobalSettings) getGlobalMetadata(esdtTokenKey []byte) (*ESDTGlobalMetadata, error) {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	val, _ := systemSCAccount.AccountDataHandler().RetrieveValue(esdtTokenKey)
	esdtMetaData := ESDTGlobalMetadataFromBytes(val)
	return &esdtMetaData, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtGlobalSettings) IsInterfaceNil() bool {
	return e == nil
}
