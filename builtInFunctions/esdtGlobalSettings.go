package builtInFunctions

import (
	"bytes"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-vm-common"
)

type esdtGlobalSettings struct {
	baseActiveHandler
	keyPrefix  []byte
	set        bool
	accounts   vmcommon.AccountsAdapter
	marshaller marshal.Marshalizer
	function   string
}

// NewESDTGlobalSettingsFunc returns the esdt pause/un-pause built-in function component
func NewESDTGlobalSettingsFunc(
	accounts vmcommon.AccountsAdapter,
	marshaller marshal.Marshalizer,
	set bool,
	function string,
	activeHandler func() bool,
) (*esdtGlobalSettings, error) {
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if activeHandler == nil {
		return nil, ErrNilActiveHandler
	}
	if !isCorrectFunction(function) {
		return nil, ErrInvalidArguments
	}

	e := &esdtGlobalSettings{
		keyPrefix:  []byte(baseESDTKeyPrefix),
		set:        set,
		accounts:   accounts,
		marshaller: marshaller,
		function:   function,
	}

	e.baseActiveHandler.activeHandler = activeHandler

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

// IsSenderOrDestinationWithTransferRole returns true if we have transfer role on the system account
func (e *esdtGlobalSettings) IsSenderOrDestinationWithTransferRole(sender, destination, tokenID []byte) bool {
	if !e.activeHandler() {
		return false
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return false
	}

	esdtTokenTransferRoleKey := append(transferAddressesKeyPrefix, tokenID...)
	addresses, _, err := getESDTRolesForAcnt(e.marshaller, systemAcc, esdtTokenTransferRoleKey)
	if err != nil {
		return false
	}

	for _, address := range addresses.Roles {
		if bytes.Equal(address, sender) || bytes.Equal(address, destination) {
			return true
		}
	}

	return false
}

func (e *esdtGlobalSettings) getGlobalMetadata(esdtTokenKey []byte) (*ESDTGlobalMetadata, error) {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	val, _, _ := systemSCAccount.AccountDataHandler().RetrieveValue(esdtTokenKey)
	esdtMetaData := ESDTGlobalMetadataFromBytes(val)
	return &esdtMetaData, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtGlobalSettings) IsInterfaceNil() bool {
	return e == nil
}
