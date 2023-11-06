package builtInFunctions

import (
	"bytes"
	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-vm-common-go"
)

type esdtSetTokenType struct {
	baseActiveHandler
	accounts   vmcommon.AccountsAdapter
	marshaller marshal.Marshalizer
}

// NewESDTSetTokenTypeFunc returns the esdt set token type built-in function component
func NewESDTSetTokenTypeFunc(
	accounts vmcommon.AccountsAdapter,
	marshaller marshal.Marshalizer,
	activeHandler func() bool,
) (*esdtSetTokenType, error) {
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if activeHandler == nil {
		return nil, ErrNilActiveHandler
	}

	e := &esdtSetTokenType{
		accounts:   accounts,
		marshaller: marshaller,
	}

	e.baseActiveHandler.activeHandler = activeHandler

	return e, nil
}

// ProcessBuiltinFunction saves the token type in the system account
func (e *esdtSetTokenType) ProcessBuiltinFunction(_, _ vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
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

	esdtTokenKey := append([]byte(baseESDTKeyPrefix), vmInput.Arguments[0]...)
	systemAccount, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	val, _, err := systemAccount.AccountDataHandler().RetrieveValue(esdtTokenKey)
	if core.IsGetNodeFromDBError(err) {
		return nil, err
	}
	esdtMetaData := ESDTGlobalMetadataFromBytes(val)
	tokenType, err := core.ConvertESDTTypeToUint32(string(vmInput.Arguments[1]))
	if err != nil {
		return nil, err
	}
	esdtMetaData.TokenType = byte(tokenType)

	err = systemAccount.AccountDataHandler().SaveKeyValue(esdtTokenKey, esdtMetaData.ToBytes())
	if err != nil {
		return nil, err
	}

	err = e.accounts.SaveAccount(systemAccount)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	return vmOutput, nil
}

func (e *esdtSetTokenType) getSystemAccount() (vmcommon.UserAccountHandler, error) {
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

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtSetTokenType) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// IsInterfaceNil returns true if there is no value under the interface
func (e *esdtSetTokenType) IsInterfaceNil() bool {
	return e == nil
}
