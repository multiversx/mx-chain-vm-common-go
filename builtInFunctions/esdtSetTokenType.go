package builtInFunctions

import (
	"bytes"
	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-vm-common-go"
)

const (
	tokenIDindex   = 0
	tokenTypeIndex = 1
)

type esdtSetTokenType struct {
	baseActiveHandler
	globalSettingsHandler vmcommon.GlobalMetadataHandler
	accounts              vmcommon.AccountsAdapter
	marshaller            marshal.Marshalizer
}

// NewESDTSetTokenTypeFunc returns the esdt set token type built-in function component
func NewESDTSetTokenTypeFunc(
	accounts vmcommon.AccountsAdapter,
	globalSettingsHandler vmcommon.GlobalMetadataHandler,
	marshaller marshal.Marshalizer,
	activeHandler func() bool,
) (*esdtSetTokenType, error) {
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if activeHandler == nil {
		return nil, ErrNilActiveHandler
	}

	e := &esdtSetTokenType{
		accounts:              accounts,
		globalSettingsHandler: globalSettingsHandler,
		marshaller:            marshaller,
	}

	e.baseActiveHandler.activeHandler = activeHandler

	return e, nil
}

// ProcessBuiltinFunction saves the token type in the system account
func (e *esdtSetTokenType) ProcessBuiltinFunction(
	_, dstAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 2 {
		return nil, ErrInvalidArguments
	}
	if !bytes.Equal(vmInput.CallerAddr, core.ESDTSCAddress) {
		return nil, ErrAddressIsNotESDTSystemSC
	}
	if !vmcommon.IsSystemAccountAddress(vmInput.RecipientAddr) {
		return nil, ErrOnlySystemAccountAccepted
	}

	esdtTokenKey := append([]byte(baseESDTKeyPrefix), vmInput.Arguments[tokenIDindex]...)
	tokenType, err := core.ConvertESDTTypeToUint32(string(vmInput.Arguments[tokenTypeIndex]))
	if err != nil {
		return nil, err
	}

	systemSCAccount, err := getSystemAccountIfNeeded(vmInput, dstAccount, e.accounts)
	if err != nil {
		return nil, err
	}

	err = e.globalSettingsHandler.SetTokenType(esdtTokenKey, tokenType, systemSCAccount)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	return vmOutput, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtSetTokenType) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// IsInterfaceNil returns true if there is no value under the interface
func (e *esdtSetTokenType) IsInterfaceNil() bool {
	return e == nil
}
