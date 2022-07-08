package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestESDTGlobalSettingsPause_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, true, core.BuiltInFunctionESDTPause, 0, &mock.EpochNotifierStub{})
	_, err := globalSettingsFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(1),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	input.CallValue = big.NewInt(0)
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input.Arguments = [][]byte{key}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotESDTSystemSC)

	input.CallerAddr = core.ESDTSCAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrOnlySystemAccountAccepted)

	input.RecipientAddr = vmcommon.SystemAccountAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	pauseKey := []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + string(key))
	assert.True(t, globalSettingsFunc.IsPaused(pauseKey))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(pauseKey))

	esdtGlobalSettingsFalse, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, false, core.BuiltInFunctionESDTUnPause, 0, &mock.EpochNotifierStub{})

	_, err = esdtGlobalSettingsFalse.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	assert.False(t, globalSettingsFunc.IsPaused(pauseKey))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(pauseKey))
}

func TestESDTGlobalSettingsLimitedTransfer_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, true, core.BuiltInFunctionESDTSetLimitedTransfer, 0, &mock.EpochNotifierStub{})
	_, err := globalSettingsFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(1),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	input.CallValue = big.NewInt(0)
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input.Arguments = [][]byte{key}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotESDTSystemSC)

	input.CallerAddr = core.ESDTSCAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrOnlySystemAccountAccepted)

	input.RecipientAddr = vmcommon.SystemAccountAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	tokenID := []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + string(key))
	assert.False(t, globalSettingsFunc.IsPaused(tokenID))
	assert.True(t, globalSettingsFunc.IsLimitedTransfer(tokenID))

	pauseFunc, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, true, core.BuiltInFunctionESDTPause, 0, &mock.EpochNotifierStub{})

	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)
	assert.True(t, globalSettingsFunc.IsPaused(tokenID))
	assert.True(t, globalSettingsFunc.IsLimitedTransfer(tokenID))

	esdtGlobalSettingsFalse, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, false, core.BuiltInFunctionESDTUnSetLimitedTransfer, 0, &mock.EpochNotifierStub{})

	_, err = esdtGlobalSettingsFalse.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
}

func TestESDTGlobalSettingsBurnForAll_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, true, vmcommon.BuiltInFunctionESDTSetBurnRoleForAll, 0, &mock.EpochNotifierStub{})
	_, err := globalSettingsFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(1),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	input.CallValue = big.NewInt(0)
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input.Arguments = [][]byte{key}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotESDTSystemSC)

	input.CallerAddr = core.ESDTSCAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrOnlySystemAccountAccepted)

	input.RecipientAddr = vmcommon.SystemAccountAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	tokenID := []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + string(key))
	assert.False(t, globalSettingsFunc.IsPaused(tokenID))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
	assert.True(t, globalSettingsFunc.IsBurnForAll(tokenID))

	pauseFunc, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, true, core.BuiltInFunctionESDTPause, 0, &mock.EpochNotifierStub{})

	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)
	assert.True(t, globalSettingsFunc.IsPaused(tokenID))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
	assert.True(t, globalSettingsFunc.IsBurnForAll(tokenID))

	esdtGlobalSettingsFalse, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, false, vmcommon.BuiltInFunctionESDTUnSetBurnRoleForAll, 0, &mock.EpochNotifierStub{})

	_, err = esdtGlobalSettingsFalse.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
}
