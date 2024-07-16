package builtInFunctions

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-vm-common-go"
)

// ESDTTypeForGlobalSettingsHandler is needed because if 0 is retrieved from the global settings handler,
// it means either that the type is not set or that the type is fungible. This will solve the ambiguity.
type ESDTTypeForGlobalSettingsHandler uint32

const (
	notSet ESDTTypeForGlobalSettingsHandler = iota
	fungible
	nonFungible
	nonFungibleV2
	metaFungible
	semiFungible
	dynamicNFT
	dynamicSFT
	dynamicMeta
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
	systemSCAccount, err := getSystemAccount(e.accounts)
	if err != nil {
		return err
	}

	esdtMetaData, err := e.GetGlobalMetadata(esdtTokenKey)
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

	return e.accounts.SaveAccount(systemSCAccount)
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

	tokenType, err := convertToESDTTokenType(uint32(esdtMetaData.TokenType))
	if errors.Is(err, ErrTypeNotSetInsideGlobalSettingsHandler) {
		return uint32(core.NonFungible), nil
	}
	if err != nil {
		return 0, err
	}

	return tokenType, nil
}

// SetTokenType sets the token type for the esdtTokenKey
func (e *esdtGlobalSettings) SetTokenType(esdtTokenKey []byte, tokenType uint32) error {
	globalSettingsTokenType, err := convertToGlobalSettingsHandlerTokenType(tokenType)
	if err != nil {
		return err
	}

	systemAccount, err := getSystemAccount(e.accounts)
	if err != nil {
		return err
	}

	val, _, err := systemAccount.AccountDataHandler().RetrieveValue(esdtTokenKey)
	if core.IsGetNodeFromDBError(err) {
		return err
	}
	esdtMetaData := ESDTGlobalMetadataFromBytes(val)
	esdtMetaData.TokenType = byte(globalSettingsTokenType)

	err = systemAccount.AccountDataHandler().SaveKeyValue(esdtTokenKey, esdtMetaData.ToBytes())
	if err != nil {
		return err
	}

	return e.accounts.SaveAccount(systemAccount)
}

func convertToGlobalSettingsHandlerTokenType(esdtType uint32) (uint32, error) {
	switch esdtType {
	case uint32(core.Fungible):
		return uint32(fungible), nil
	case uint32(core.NonFungible):
		return uint32(nonFungible), nil
	case uint32(core.NonFungibleV2):
		return uint32(nonFungibleV2), nil
	case uint32(core.MetaFungible):
		return uint32(metaFungible), nil
	case uint32(core.SemiFungible):
		return uint32(semiFungible), nil
	case uint32(core.DynamicNFT):
		return uint32(dynamicNFT), nil
	case uint32(core.DynamicSFT):
		return uint32(dynamicSFT), nil
	case uint32(core.DynamicMeta):
		return uint32(dynamicMeta), nil
	default:
		return math.MaxUint32, fmt.Errorf("invalid esdt type: %d", esdtType)
	}
}

func convertToESDTTokenType(esdtType uint32) (uint32, error) {
	switch ESDTTypeForGlobalSettingsHandler(esdtType) {
	case notSet:
		return 0, ErrTypeNotSetInsideGlobalSettingsHandler
	case fungible:
		return uint32(core.Fungible), nil
	case nonFungible:
		return uint32(core.NonFungible), nil
	case nonFungibleV2:
		return uint32(core.NonFungibleV2), nil
	case metaFungible:
		return uint32(core.MetaFungible), nil
	case semiFungible:
		return uint32(core.SemiFungible), nil
	case dynamicNFT:
		return uint32(core.DynamicNFT), nil
	case dynamicSFT:
		return uint32(core.DynamicSFT), nil
	case dynamicMeta:
		return uint32(core.DynamicMeta), nil
	default:
		return math.MaxUint32, fmt.Errorf("invalid esdt type: %d", esdtType)
	}
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtGlobalSettings) IsInterfaceNil() bool {
	return e == nil
}
