package builtInFunctions

import (
	"bytes"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-vm-common-go"
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
	_, dstAccount vmcommon.UserAccountHandler,
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

	systemSCAccount, err := e.getSystemAccountIfNeeded(vmInput, dstAccount)
	if err != nil {
		return nil, err
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	err = e.toggleSetting(esdtTokenKey, systemSCAccount)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	return vmOutput, nil
}

func (e *esdtGlobalSettings) getSystemAccountIfNeeded(
	vmInput *vmcommon.ContractCallInput,
	dstAccount vmcommon.UserAccountHandler,
) (vmcommon.UserAccountHandler, error) {
	if !bytes.Equal(core.SystemAccountAddress, vmInput.RecipientAddr) || check.IfNil(dstAccount) {
		return getSystemAccount(e.accounts)
	}

	return dstAccount, nil
}

func (e *esdtGlobalSettings) toggleSetting(esdtTokenKey []byte, systemSCAccount vmcommon.UserAccountHandler) error {
	esdtMetaData, err := e.getGlobalMetadataFromAccount(esdtTokenKey, systemSCAccount)
	if err != nil {
		return err
	}

	switch e.function {
	case core.BuiltInFunctionESDTSetLimitedTransfer, core.BuiltInFunctionESDTUnSetLimitedTransfer:
		esdtMetaData.LimitedTransfer = e.set
	case core.BuiltInFunctionESDTPause, core.BuiltInFunctionESDTUnPause:
		esdtMetaData.Paused = e.set
	case vmcommon.BuiltInFunctionESDTUnSetBurnRoleForAll, vmcommon.BuiltInFunctionESDTSetBurnRoleForAll:
		esdtMetaData.BurnRoleForAll = e.set
	}

	err = systemSCAccount.AccountDataHandler().SaveKeyValue(esdtTokenKey, esdtMetaData.ToBytes())
	if err != nil {
		return err
	}

	log.Error("are we here ?")
	return nil
}

func getSystemAccount(accounts vmcommon.AccountsAdapter) (vmcommon.UserAccountHandler, error) {
	systemSCAccount, err := accounts.LoadAccount(vmcommon.SystemAccountAddress)
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
	esdtMetadata, err := e.GetGlobalMetadata(esdtTokenKey)
	if err != nil {
		return false
	}

	return esdtMetadata.Paused
}

// IsLimitedTransfer returns true if the esdtTokenKey (prefixed) is with limited transfer
func (e *esdtGlobalSettings) IsLimitedTransfer(esdtTokenKey []byte) bool {
	esdtMetadata, err := e.GetGlobalMetadata(esdtTokenKey)
	if err != nil {
		return false
	}

	return esdtMetadata.LimitedTransfer
}

// IsBurnForAll returns true if the esdtTokenKey (prefixed) is with burn for all
func (e *esdtGlobalSettings) IsBurnForAll(esdtTokenKey []byte) bool {
	esdtMetadata, err := e.GetGlobalMetadata(esdtTokenKey)
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

	systemAcc, err := getSystemAccount(e.accounts)
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

// GetGlobalMetadata returns the global metadata for the esdtTokenKey
func (e *esdtGlobalSettings) GetGlobalMetadata(esdtTokenKey []byte) (*ESDTGlobalMetadata, error) {
	systemSCAccount, err := getSystemAccount(e.accounts)
	if err != nil {
		return nil, err
	}

	return e.getGlobalMetadataFromAccount(esdtTokenKey, systemSCAccount)
}

func (e *esdtGlobalSettings) getGlobalMetadataFromAccount(
	esdtTokenKey []byte,
	systemSCAccount vmcommon.UserAccountHandler,
) (*ESDTGlobalMetadata, error) {
	val, _, err := systemSCAccount.AccountDataHandler().RetrieveValue(esdtTokenKey)
	if core.IsGetNodeFromDBError(err) {
		return nil, err
	}
	esdtMetaData := ESDTGlobalMetadataFromBytes(val)
	return &esdtMetaData, nil
}

// GetTokenType returns the token type for the esdtTokenKey
func (e *esdtGlobalSettings) GetTokenType(esdtTokenKey []byte) (uint32, error) {
	esdtMetaData, err := e.GetGlobalMetadata(esdtTokenKey)
	if err != nil {
		return 0, err
	}

	return uint32(esdtMetaData.TokenType), nil
}

// SetTokenType sets the token type for the esdtTokenKey
func (e *esdtGlobalSettings) SetTokenType(esdtTokenKey []byte, tokenType uint32) error {
	systemAccount, err := getSystemAccount(e.accounts)
	if err != nil {
		return err
	}

	val, _, err := systemAccount.AccountDataHandler().RetrieveValue(esdtTokenKey)
	if core.IsGetNodeFromDBError(err) {
		return err
	}
	esdtMetaData := ESDTGlobalMetadataFromBytes(val)
	esdtMetaData.TokenType = byte(tokenType)

	err = systemAccount.AccountDataHandler().SaveKeyValue(esdtTokenKey, esdtMetaData.ToBytes())
	if err != nil {
		return err
	}

	return e.accounts.SaveAccount(systemAccount)
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtGlobalSettings) IsInterfaceNil() bool {
	return e == nil
}
