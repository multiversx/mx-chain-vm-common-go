package builtInFunctions

import (
	"errors"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewESDTGlobalSettingsFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil accounts should error", func(t *testing.T) {
		t.Parallel()

		globalSettingsFunc, err := NewESDTGlobalSettingsFunc(nil, &mock.MarshalizerMock{}, true, core.BuiltInFunctionESDTPause, trueHandler)
		assert.Equal(t, ErrNilAccountsAdapter, err)
		assert.True(t, check.IfNil(globalSettingsFunc))
	})
	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		globalSettingsFunc, err := NewESDTGlobalSettingsFunc(&mock.AccountsStub{}, nil, true, core.BuiltInFunctionESDTPause, trueHandler)
		assert.Equal(t, ErrNilMarshalizer, err)
		assert.True(t, check.IfNil(globalSettingsFunc))
	})
	t.Run("nil active handler should error", func(t *testing.T) {
		t.Parallel()

		globalSettingsFunc, err := NewESDTGlobalSettingsFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionESDTPause, nil)
		assert.Equal(t, ErrNilActiveHandler, err)
		assert.True(t, check.IfNil(globalSettingsFunc))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		globalSettingsFunc, err := NewESDTGlobalSettingsFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionESDTPause, falseHandler)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(globalSettingsFunc))
	})
}

func TestESDTGlobalSettingsPause_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionESDTPause, falseHandler)
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

	pauseKey := []byte(baseESDTKeyPrefix + string(key))
	assert.True(t, globalSettingsFunc.IsPaused(pauseKey))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(pauseKey))

	esdtGlobalSettingsFalse, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, false, core.BuiltInFunctionESDTUnPause, falseHandler)

	_, err = esdtGlobalSettingsFalse.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	assert.False(t, globalSettingsFunc.IsPaused(pauseKey))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(pauseKey))
}

func TestESDTGlobalSettingsPause_ProcessBuiltInFunctionGetNodeFromDbErr(t *testing.T) {
	t.Parallel()

	globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(
		&mock.AccountsStub{
			LoadAccountCalled: func(_ []byte) (vmcommon.AccountHandler, error) {
				return &mock.AccountWrapMock{
					RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
						return nil, 0, core.NewGetNodeFromDBErrWithKey([]byte("key"), errors.New("error"), "")
					},
				}, nil
			},
		},
		&mock.MarshalizerMock{},
		true,
		core.BuiltInFunctionESDTPause,
		falseHandler,
	)

	key := []byte("key")
	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{key},
			CallerAddr:  core.ESDTSCAddress,
		},
		RecipientAddr: vmcommon.SystemAccountAddress,
	}

	output, err := globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, output)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestESDTGlobalSettingsLimitedTransfer_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionESDTSetLimitedTransfer, trueHandler)
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

	tokenID := []byte(baseESDTKeyPrefix + string(key))
	assert.False(t, globalSettingsFunc.IsPaused(tokenID))
	assert.True(t, globalSettingsFunc.IsLimitedTransfer(tokenID))

	pauseFunc, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionESDTPause, falseHandler)

	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)
	assert.True(t, globalSettingsFunc.IsPaused(tokenID))
	assert.True(t, globalSettingsFunc.IsLimitedTransfer(tokenID))

	esdtGlobalSettingsFalse, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, false, core.BuiltInFunctionESDTUnSetLimitedTransfer, trueHandler)

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
	}, &mock.MarshalizerMock{}, true, vmcommon.BuiltInFunctionESDTSetBurnRoleForAll, falseHandler)
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

	tokenID := []byte(baseESDTKeyPrefix + string(key))
	assert.False(t, globalSettingsFunc.IsPaused(tokenID))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
	assert.True(t, globalSettingsFunc.IsBurnForAll(tokenID))

	pauseFunc, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionESDTPause, falseHandler)

	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)
	assert.True(t, globalSettingsFunc.IsPaused(tokenID))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
	assert.True(t, globalSettingsFunc.IsBurnForAll(tokenID))

	esdtGlobalSettingsFalse, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, false, vmcommon.BuiltInFunctionESDTUnSetBurnRoleForAll, falseHandler)

	_, err = esdtGlobalSettingsFalse.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
}

func TestEsdtGlobalSettings_SetTokenType(t *testing.T) {
	t.Parallel()

	t.Run("invalid token type", func(t *testing.T) {
		t.Parallel()

		acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
		globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(
			&mock.AccountsStub{
				LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
					return acnt, nil
				},
			},
			&mock.MarshalizerMock{},
			true,
			core.BuiltInFunctionESDTPause,
			falseHandler,
		)

		err := globalSettingsFunc.SetTokenType([]byte("key"), 100)
		require.True(t, strings.Contains(err.Error(), "invalid esdt type"))
	})
	t.Run("fungible token type", func(t *testing.T) {
		t.Parallel()

		acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
		globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(
			&mock.AccountsStub{
				LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
					return acnt, nil
				},
			},
			&mock.MarshalizerMock{},
			true,
			core.BuiltInFunctionESDTPause,
			falseHandler,
		)

		err := globalSettingsFunc.SetTokenType([]byte("key"), uint32(core.Fungible))
		require.Nil(t, err)
		retrievedVal := acnt.Storage["key"]
		require.Equal(t, []byte{0, 1}, retrievedVal)
	})
}

func TestEsdtGlobalSettings_GetTokenType(t *testing.T) {
	t.Parallel()

	t.Run("token type not set", func(t *testing.T) {
		t.Parallel()

		acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
		globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(
			&mock.AccountsStub{
				LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
					return acnt, nil
				},
			},
			&mock.MarshalizerMock{},
			true,
			core.BuiltInFunctionESDTPause,
			falseHandler,
		)

		acnt.Storage["key"] = []byte{byte(notSet)}
		val, err := globalSettingsFunc.GetTokenType([]byte("key"))
		require.Nil(t, err)
		require.Equal(t, uint32(core.NonFungible), val)
	})
	t.Run("retrieve token type error ", func(t *testing.T) {
		t.Parallel()

		acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
		globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(
			&mock.AccountsStub{
				LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
					return acnt, nil
				},
			},
			&mock.MarshalizerMock{},
			true,
			core.BuiltInFunctionESDTPause,
			falseHandler,
		)

		acnt.Storage["key"] = []byte{0, 100}
		val, err := globalSettingsFunc.GetTokenType([]byte("key"))
		require.True(t, strings.Contains(err.Error(), "invalid esdt type"))
		require.Equal(t, uint32(0), val)
	})
	t.Run("convert to esdt token type error", func(t *testing.T) {
		t.Parallel()

		acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
		globalSettingsFunc, _ := NewESDTGlobalSettingsFunc(
			&mock.AccountsStub{
				LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
					return acnt, nil
				},
			},
			&mock.MarshalizerMock{},
			true,
			core.BuiltInFunctionESDTPause,
			falseHandler,
		)

		acnt.Storage["key"] = []byte{0, byte(fungible)}
		val, err := globalSettingsFunc.GetTokenType([]byte("key"))
		require.Nil(t, err)
		require.Equal(t, uint32(core.Fungible), val)
	})
}
