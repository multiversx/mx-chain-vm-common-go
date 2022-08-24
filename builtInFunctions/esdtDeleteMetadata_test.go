package builtInFunctions

import (
	"bytes"
	"errors"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func createMockArgsForNewESDTDelete() ArgsNewESDTDeleteMetadata {
	return ArgsNewESDTDeleteMetadata{
		FuncGasCost:      1,
		Marshalizer:      &mock.MarshalizerMock{},
		Accounts:         &mock.AccountsStub{},
		ActivationEpoch:  0,
		EpochNotifier:    &mock.EpochNotifierStub{},
		AllowedAddresses: [][]byte{bytes.Repeat([]byte{1}, 32)},
		Delete:           true,
	}
}

func TestNewESDTDeleteMetadataFunc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDelete()
	args.Marshalizer = nil
	_, err := NewESDTDeleteMetadataFunc(args)
	assert.Equal(t, err, ErrNilMarshalizer)

	args = createMockArgsForNewESDTDelete()
	args.Accounts = nil
	_, err = NewESDTDeleteMetadataFunc(args)
	assert.Equal(t, err, ErrNilAccountsAdapter)

	args = createMockArgsForNewESDTDelete()
	args.EpochNotifier = nil
	_, err = NewESDTDeleteMetadataFunc(args)
	assert.Equal(t, err, ErrNilEpochHandler)

	args = createMockArgsForNewESDTDelete()
	e, err := NewESDTDeleteMetadataFunc(args)
	assert.Nil(t, err)
	assert.False(t, e.IsInterfaceNil())
	assert.True(t, e.IsActive())

	args = createMockArgsForNewESDTDelete()
	args.ActivationEpoch = 1
	e, _ = NewESDTDeleteMetadataFunc(args)
	assert.False(t, e.IsActive())

	e.SetNewGasConfig(&vmcommon.GasCost{})
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
	vmInput.CallerAddr = e.allowedAddresses[0]
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidRcvAddr)

	vmInput.RecipientAddr = e.allowedAddresses[0]
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
	vmInput.CallerAddr = e.allowedAddresses[0]
	vmInput.RecipientAddr = e.allowedAddresses[0]
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
	vmInput.CallerAddr = args.AllowedAddresses[0]
	vmInput.RecipientAddr = args.AllowedAddresses[0]
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

func TestEsdtDeleteMetaData_ProcessBuiltinFunctionDeleteAllowedAddresses(t *testing.T) {
	t.Parallel()

	allowedAddress1 := bytes.Repeat([]byte{1}, 32)
	allowedAddress2 := bytes.Repeat([]byte{2}, 32)
	allowedAddress3 := bytes.Repeat([]byte{3}, 32)
	notAllowedAddress := bytes.Repeat([]byte{4}, 32)

	args := createMockArgsForNewESDTDelete()
	args.AllowedAddresses = [][]byte{
		allowedAddress1,
		allowedAddress2,
		allowedAddress3,
	}

	e, _ := NewESDTDeleteMetadataFunc(args)
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		}}

	e.accounts = accounts

	tokenName := []byte("TOKEN-ababab")
	nrIntervals := []byte{1}
	interval0Start := []byte{1}
	interval0End := []byte{10}

	arguments := [][]byte{
		tokenName,
		nrIntervals,
		interval0Start,
		interval0End,
	}

	var tests = []struct {
		name          string
		address       []byte
		expectedError error
	}{
		{"allowAddress 1", allowedAddress1, nil},
		{"allowAddress 2", allowedAddress2, nil},
		{"allowAddress 3", allowedAddress3, nil},
		{"not allowed Address", notAllowedAddress, ErrAddressIsNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vmInput := &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}}
			vmInput.CallerAddr = tt.address
			vmInput.RecipientAddr = tt.address
			vmInput.Arguments = arguments

			_, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
			assert.Equal(t, tt.expectedError, err)
		})
	}
}
