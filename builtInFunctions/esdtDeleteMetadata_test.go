package builtInFunctions

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func createMockArgsForNewESDTDelete() ArgsNewESDTDeleteMetadata {
	return ArgsNewESDTDeleteMetadata{
		FuncGasCost:    1,
		Marshalizer:    &mock.MarshalizerMock{},
		Accounts:       &mock.AccountsStub{},
		AllowedAddress: bytes.Repeat([]byte{1}, 32),
		Delete:         true,
		ActiveHandler:  trueHandler,
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
		args.ActiveHandler = nil
		_, err := NewESDTDeleteMetadataFunc(args)
		assert.Equal(t, err, ErrNilActiveHandler)
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
	assert.Equal(t, err, ErrNilVmInput)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(10)}})
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}})
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrAddressIsNotAllowed)

	vmInput := &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}}
	vmInput.CallerAddr = e.allowedAddress
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidRcvAddr)

	vmInput.RecipientAddr = e.allowedAddress
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNumOfArgs)

	e.delete = false
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNumOfArgs)

	vmInput.Arguments = [][]byte{{1}, {0}, {1}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.NotNil(t, err)

	e.delete = true
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNumOfArgs)

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
	assert.Equal(t, err, ErrInvalidNonce)

	vmInput.Arguments[0] = []byte("TOKEN-ABABAB")
	vmInput.Arguments[1] = []byte{1}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidTokenID)

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
	assert.NotNil(t, ErrTokenHasValidMetadata)

	_ = acnt.SaveKeyValue(esdtNftTokenKey, nil)
	testErr := errors.New("testError")
	accounts.SaveAccountCalled = func(account vmcommon.AccountHandler) error {
		return testErr
	}

	vmInput.Arguments[1] = []byte{2}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidMetadata)

	vmInput.Arguments[1] = []byte{1}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, testErr)
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
	assert.Equal(t, err, ErrInvalidNumOfArgs)

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
	assert.Equal(t, err, ErrInvalidNumOfArgs)

	vmInput.Arguments = append(vmInput.Arguments, []byte{1}, []byte{2}, []byte{4}, []byte{10})
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.NotNil(t, vmOutput)
}
