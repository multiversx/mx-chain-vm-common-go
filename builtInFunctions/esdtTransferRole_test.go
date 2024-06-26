package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewESDTTransferRoleAddressFunc(t *testing.T) {
	_, err := NewESDTTransferRoleAddressFunc(nil, &mock.MarshalizerMock{}, 10, true, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SendAlwaysFlag
		},
	})
	assert.Equal(t, err, ErrNilAccountsAdapter)

	_, err = NewESDTTransferRoleAddressFunc(&mock.AccountsStub{}, nil, 10, true, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SendAlwaysFlag
		},
	})
	assert.Equal(t, err, ErrNilMarshalizer)

	e, err := NewESDTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 0, true, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SendAlwaysFlag
		},
	})
	assert.Equal(t, err, ErrInvalidMaxNumAddresses)

	_, err = NewESDTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 10, true, nil)
	assert.Equal(t, err, ErrNilEnableEpochsHandler)
	assert.True(t, check.IfNil(e))

	e, err = NewESDTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 10, true, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SendAlwaysFlag
		},
	})
	assert.Nil(t, err)

	e.SetNewGasConfig(nil)
	assert.False(t, e.IsInterfaceNil())
}

func TestESDTTransferRoleProcessBuiltInFunction_Errors(t *testing.T) {
	accounts := &mock.AccountsStub{}
	marshaller := &mock.MarshalizerMock{}
	e, err := NewESDTTransferRoleAddressFunc(accounts, marshaller, 10, true, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SendAlwaysFlag
		},
	})
	assert.Nil(t, err)

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

	errNotImplemented := errors.New("not implemented")
	vmInput.RecipientAddr = vmcommon.SystemAccountAddress
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, errNotImplemented)

	systemAcc := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		return systemAcc, nil
	}
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
	e, err := NewESDTTransferRoleAddressFunc(accounts, marshaller, 10, true, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SendAlwaysFlag
		},
	})
	assert.Nil(t, err)

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

func TestGetEsdtRolesForAcnt(t *testing.T) {
	t.Parallel()

	acc := &mock.AccountWrapMock{
		RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
			return nil, 0, core.NewGetNodeFromDBErrWithKey([]byte("key"), errors.New("error"), "")
		},
	}
	addresses, _, err := getESDTRolesForAcnt(&mock.MarshalizerMock{}, acc, []byte("key"))
	assert.Nil(t, addresses)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestESDTTransferRoleIsSenderOrDestinationWithTransferRole(t *testing.T) {
	accounts := &mock.AccountsStub{}
	marshaller := &mock.MarshalizerMock{}
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SendAlwaysFlag
		},
	}
	e, err := NewESDTTransferRoleAddressFunc(accounts, marshaller, 10, true, enableEpochsHandler)
	assert.Nil(t, err)

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

	globalSettings, _ := NewESDTGlobalSettingsFunc(
		accounts,
		marshaller,
		true,
		vmcommon.BuiltInFunctionESDTSetBurnRoleForAll,
		func() bool {
			return enableEpochsHandler.IsFlagEnabledCalled(SendAlwaysFlag)
		},
	)
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
