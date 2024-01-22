package builtInFunctions

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func createMockArgsForNewESDTDelete() ArgsNewESDTDeleteMetadata {
	return ArgsNewESDTDeleteMetadata{
		FuncGasCost:    1,
		Marshalizer:    &mock.MarshalizerMock{},
		Accounts:       &mock.AccountsStub{},
		AllowedAddress: bytes.Repeat([]byte{1}, 32),
		Delete:         true,
		EnableEpochsHandler: &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SendAlwaysFlag
			},
		},
	}
}

func TestNewESDTDeleteMetadataFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsForNewESDTDelete()
		args.Marshalizer = nil
		_, err := NewESDTDeleteMetadataFunc(args)
		assert.Equal(t, err, ErrNilMarshalizer)
	})
	t.Run("nil accounts adapter should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsForNewESDTDelete()
		args.Accounts = nil
		_, err := NewESDTDeleteMetadataFunc(args)
		assert.Equal(t, err, ErrNilAccountsAdapter)
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsForNewESDTDelete()
		args.EnableEpochsHandler = nil
		_, err := NewESDTDeleteMetadataFunc(args)
		assert.Equal(t, err, ErrNilEnableEpochsHandler)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsForNewESDTDelete()
		e, err := NewESDTDeleteMetadataFunc(args)
		assert.Nil(t, err)
		assert.False(t, e.IsInterfaceNil())
		assert.True(t, e.IsActive())
	})
}

func TestEsdtDeleteMetaData_ProcessBuiltinFunctionErrors(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDelete()
	e, _ := NewESDTDeleteMetadataFunc(args)

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilVmInput, err)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(10)}})
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrBuiltInFunctionCalledWithValue, err)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}})
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrAddressIsNotAllowed, err)

	vmInput := &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}}
	vmInput.CallerAddr = e.allowedAddress
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidRcvAddr, err)

	vmInput.RecipientAddr = e.allowedAddress
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidNumOfArgs, err)

	e.delete = false
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidNumOfArgs, err)

	vmInput.Arguments = [][]byte{{1}, {0}, {1}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.NotNil(t, err)

	e.delete = true
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidNumOfArgs, err)

	vmInput.Arguments = [][]byte{{1}, {0}, {1}, {1}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.NotNil(t, err)

	e.delete = false

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
		return acnt, nil
	}}

	e.accounts = accounts
	vmInput.Arguments = [][]byte{{1}, {0}, {1}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidNonce, err)

	vmInput.Arguments[0] = []byte("TOKEN-ABABAB")
	vmInput.Arguments[1] = []byte{1}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidTokenID, err)

	vmInput.Arguments[0] = []byte("TOKEN-ababab")
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.NotNil(t, err)

	esdtMetadata := &esdt.MetaData{Name: []byte("something"), Nonce: 1}
	marshalledData, _ := e.marshaller.Marshal(esdtMetadata)
	vmInput.Arguments[2] = make([]byte, len(marshalledData))
	copy(vmInput.Arguments[2], marshalledData)

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	esdtNftTokenKey := computeESDTNFTTokenKey(esdtTokenKey, 1)
	err = acnt.SaveKeyValue(esdtNftTokenKey, []byte("t"))
	assert.Nil(t, err)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.NotNil(t, err)

	esdtData := &esdt.ESDigitalToken{Value: big.NewInt(0), TokenMetaData: &esdt.MetaData{Name: []byte("data")}}
	marshalledData, _ = e.marshaller.Marshal(esdtData)
	err = acnt.SaveKeyValue(esdtNftTokenKey, marshalledData)
	assert.Nil(t, err)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrTokenHasValidMetadata, err)

	_ = acnt.SaveKeyValue(esdtNftTokenKey, nil)
	testErr := errors.New("testError")
	accounts.SaveAccountCalled = func(account vmcommon.AccountHandler) error {
		return testErr
	}

	vmInput.Arguments[1] = []byte{2}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidMetadata, err)

	vmInput.Arguments[1] = []byte{1}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, testErr, err)
}

func TestEsdtDeleteMetaData_ProcessBuiltinFunctionGetNodeFromDbErr(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDelete()
	args.Delete = false
	args.Accounts = &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return &mock.AccountWrapMock{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, core.NewGetNodeFromDBErrWithKey([]byte("key"), errors.New("error"), "")
				},
			}, nil
		},
	}
	esdtMetadata := &esdt.MetaData{Name: []byte("something"), Nonce: 1}
	marshalledData, _ := args.Marshalizer.Marshal(esdtMetadata)
	e, _ := NewESDTDeleteMetadataFunc(args)
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: e.allowedAddress,
			Arguments:  [][]byte{[]byte("TOKEN-ababab"), {1}, marshalledData},
		},
		RecipientAddr: e.allowedAddress,
	}

	output, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, output)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestEsdtDeleteMetaData_ProcessBuiltinFunctionAdd(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDelete()
	args.Delete = false
	e, _ := NewESDTDeleteMetadataFunc(args)

	vmInput := &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}}
	vmInput.CallerAddr = e.allowedAddress
	vmInput.RecipientAddr = e.allowedAddress
	vmInput.Arguments = [][]byte{{1}, {0}, {1}}
	vmInput.Arguments[0] = []byte("TOKEN-ababab")
	vmInput.Arguments[1] = []byte{1}
	esdtMetadata := &esdt.MetaData{Name: []byte("something"), Nonce: 1}
	marshalledData, _ := e.marshaller.Marshal(esdtMetadata)
	vmInput.Arguments[2] = make([]byte, len(marshalledData))
	copy(vmInput.Arguments[2], marshalledData)

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		}}

	e.accounts = accounts

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.NotNil(t, vmOutput)
	assert.Nil(t, err)
}

func TestEsdtDeleteMetaData_ProcessBuiltinFunctionDelete(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDelete()
	e, _ := NewESDTDeleteMetadataFunc(args)

	vmInput := &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}}
	vmInput.CallerAddr = e.allowedAddress
	vmInput.RecipientAddr = e.allowedAddress
	vmInput.Arguments = [][]byte{{1}, {2}, {1}, {1}}

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		}}

	e.accounts = accounts

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidTokenID)

	vmInput.Arguments[0] = []byte("TOKEN-ababab")
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidNumOfArgs, err)

	vmInput.Arguments[2] = []byte{0}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNonce)

	vmInput.Arguments[2] = []byte{10}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidArguments)

	vmInput.Arguments[1] = []byte{1}
	vmInput.Arguments[3] = []byte{11}

	vmInput.Arguments = append(vmInput.Arguments, []byte("TOKEN-ababab"), []byte{2})
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidNumOfArgs, err)

	vmInput.Arguments = append(vmInput.Arguments, []byte{1}, []byte{2}, []byte{4}, []byte{10})
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.NotNil(t, vmOutput)
	assert.Nil(t, err)
}
