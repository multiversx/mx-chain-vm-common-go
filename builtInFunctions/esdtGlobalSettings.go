package builtInFunctions

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/atomic"
	"github.com/ElrondNetwork/elrond-vm-common/check"
)

type esdtGlobalSettings struct {
	*baseEnabled
	keyPrefix []byte
	set       bool
	function  string
	accounts  vmcommon.AccountsAdapter
}

// NewESDTGlobalSettingsFunc returns the esdt pause/un-pause built-in function component
func NewESDTGlobalSettingsFunc(
	accounts vmcommon.AccountsAdapter,
	set bool,
	function string,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*esdtGlobalSettings, error) {
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}

	e := &esdtGlobalSettings{
		keyPrefix: []byte(vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTKeyIdentifier),
		set:       set,
		accounts:  accounts,
	}

	e.baseEnabled = &baseEnabled{
		function:        function,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
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
	if !bytes.Equal(vmInput.CallerAddr, vmcommon.ESDTSCAddress) {
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

func (e *esdtGlobalSettings) toggleSetting(token []byte) error {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	val, _ := systemSCAccount.AccountDataHandler().RetrieveValue(token)
	esdtMetaData := ESDTGlobalMetadataFromBytes(val)

	switch e.function {
	case
	}

	esdtMetaData.Paused = e.pause
	err = systemSCAccount.AccountDataHandler().SaveKeyValue(token, esdtMetaData.ToBytes())
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

// IsPaused returns true if the token is paused
func (e *esdtGlobalSettings) IsPaused(pauseKey []byte) bool {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return false
	}

	val, _ := systemSCAccount.AccountDataHandler().RetrieveValue(pauseKey)
	if len(val) != lengthOfESDTMetadata {
		return false
	}
	esdtMetaData := ESDTGlobalMetadataFromBytes(val)

	return esdtMetaData.Paused
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtGlobalSettings) IsInterfaceNil() bool {
	return e == nil
}
