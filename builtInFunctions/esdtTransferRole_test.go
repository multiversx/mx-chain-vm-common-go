package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewESDTTransferRoleAddressFunc(t *testing.T) {
	_, err := NewESDTTransferRoleAddressFunc(nil, &mock.MarshalizerMock{}, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Equal(t, err, ErrNilAccountsAdapter)

	_, err = NewESDTTransferRoleAddressFunc(&mock.AccountsStub{}, nil, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Equal(t, err, ErrNilMarshalizer)

	_, err = NewESDTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 0, nil, 10, true)
	assert.Equal(t, err, ErrNilEpochHandler)

	e, err := NewESDTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 0, &mock.EpochNotifierStub{}, 0, true)
	assert.Equal(t, err, ErrInvalidMaxNumAddresses)
	assert.True(t, check.IfNil(e))

	e, err = NewESDTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Nil(t, err)
	assert.Equal(t, e.function, vmcommon.BuiltInFunctionESDTTransferRoleAddAddress)

	e, err = NewESDTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 0, &mock.EpochNotifierStub{}, 10, false)
	assert.Nil(t, err)
	assert.Equal(t, e.function, vmcommon.BuiltInFunctionESDTTransferRoleDeleteAddress)

	e.SetNewGasConfig(nil)
	assert.False(t, e.IsInterfaceNil())
}

func TestESDTTransferRoleProcessBuiltInFunction_Errors(t *testing.T) {
	accounts := &mock.AccountsStub{}
	marshaller := &mock.MarshalizerMock{}
	e, err := NewESDTTransferRoleAddressFunc(accounts, marshaller, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Nil(t, err)
	assert.Equal(t, e.function, vmcommon.BuiltInFunctionESDTTransferRoleAddAddress)

	_, err = e.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("token"), {1}, {2}, {3}},
		},
		RecipientAddr:     nil,
		Function:          "",
		AllowInitFunction: false,
	}

	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, ErrAddressIsNotESDTSystemSC)

	vmInput.CallerAddr = core.ESDTSCAddress
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, ErrOnlySystemAccountAccepted)

	vmInput.RecipientAddr = vmcommon.SystemAccountAddress
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, errors.New("not implemented"))

	systemAcc := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		return systemAcc, nil
	}
	errNotImplemented := errors.New("not implemented")
	accounts.SaveAccountCalled = func(account vmcommon.AccountHandler) error {
		return errNotImplemented
	}
	e.maxNumAddresses = 1
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, ErrTooManyTransferAddresses)

	e.maxNumAddresses = 10
	marshaller.Fail = true
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, errors.New("MarshalizerMock generic error"))

	systemAcc.Storage[string(append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))] = []byte{1, 1, 1}
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, errors.New("MarshalizerMock generic error"))

	marshaller.Fail = false
	systemAcc.Storage[string(append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))] = nil
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, errNotImplemented)
}

func TestESDTTransferRoleProcessBuiltInFunction_AddNewAddresses(t *testing.T) {
	accounts := &mock.AccountsStub{}
	marshaller := &mock.MarshalizerMock{}
	e, err := NewESDTTransferRoleAddressFunc(accounts, marshaller, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Nil(t, err)
	assert.Equal(t, e.function, vmcommon.BuiltInFunctionESDTTransferRoleAddAddress)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: core.ESDTSCAddress,
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("token"), {1}, {2}, {3}},
		},
		RecipientAddr:     vmcommon.SystemAccountAddress,
		Function:          "",
		AllowInitFunction: false,
	}

	systemAcc := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		return systemAcc, nil
	}

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)

	addresses, _, _ := getESDTRolesForAcnt(e.marshaller, systemAcc, append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))
	assert.Equal(t, len(addresses.Roles), 3)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)

	addresses, _, _ = getESDTRolesForAcnt(e.marshaller, systemAcc, append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))
	assert.Equal(t, len(addresses.Roles), 3)

	e.set = false
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)
	addresses, _, _ = getESDTRolesForAcnt(e.marshaller, systemAcc, append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))
	assert.Equal(t, len(addresses.Roles), 0)
}

func TestESDTTransferRoleIsSenderOrDestinationWithTransferRole(t *testing.T) {
	accounts := &mock.AccountsStub{}
	marshaller := &mock.MarshalizerMock{}
	e, err := NewESDTTransferRoleAddressFunc(accounts, marshaller, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Nil(t, err)
	assert.Equal(t, e.function, vmcommon.BuiltInFunctionESDTTransferRoleAddAddress)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: core.ESDTSCAddress,
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("token"), {1}, {2}, {3}},
		},
		RecipientAddr:     vmcommon.SystemAccountAddress,
		Function:          "",
		AllowInitFunction: false,
	}

	systemAcc := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		return systemAcc, nil
	}

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)

	addresses, _, _ := getESDTRolesForAcnt(e.marshaller, systemAcc, append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))
	assert.Equal(t, len(addresses.Roles), 3)

	globalSettings, _ := NewESDTGlobalSettingsFunc(accounts, marshaller, true, vmcommon.BuiltInFunctionESDTSetBurnRoleForAll, 0, &mock.EpochNotifierStub{})
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(nil, nil, nil))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], []byte("random"), []byte("random")))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], vmInput.Arguments[2], []byte("random")))
	assert.True(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], vmInput.Arguments[2], vmInput.Arguments[0]))
	assert.True(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], []byte("random"), vmInput.Arguments[0]))
	assert.True(t, globalSettings.IsSenderOrDestinationWithTransferRole([]byte("random"), vmInput.Arguments[2], vmInput.Arguments[0]))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole([]byte("random"), []byte("random"), vmInput.Arguments[0]))

	e.set = false
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)
	addresses, _, _ = getESDTRolesForAcnt(e.marshaller, systemAcc, append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))
	assert.Equal(t, len(addresses.Roles), 0)
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(nil, nil, nil))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], []byte("random"), []byte("random")))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], vmInput.Arguments[2], []byte("random")))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], vmInput.Arguments[2], vmInput.Arguments[0]))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], []byte("random"), vmInput.Arguments[0]))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole([]byte("random"), vmInput.Arguments[2], vmInput.Arguments[0]))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole([]byte("random"), []byte("random"), vmInput.Arguments[0]))
}
