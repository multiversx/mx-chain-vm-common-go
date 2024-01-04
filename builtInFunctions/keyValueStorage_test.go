package builtInFunctions

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var enabledFixForSaveKeyValueEnableEpochsHandler = &mock.EnableEpochsHandlerStub{
	IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
		return flag == FixGasRemainingForSaveKeyValueFlag
	},
}

var disabledFixForSaveKeyValueEnableEpochsHandler = &mock.EnableEpochsHandlerStub{}

func TestNewSaveKeyValueStorageFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil enable epoch hanlder should error", func(t *testing.T) {
		t.Parallel()

		funcGasCost := uint64(1)
		gasConfig := vmcommon.BaseOperationCost{
			StorePerByte: 1,
		}

		kvs, err := NewSaveKeyValueStorageFunc(gasConfig, funcGasCost, nil)
		assert.Nil(t, kvs)
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		funcGasCost := uint64(1)
		gasConfig := vmcommon.BaseOperationCost{
			StorePerByte: 1,
		}

		kvs, err := NewSaveKeyValueStorageFunc(gasConfig, funcGasCost, disabledFixForSaveKeyValueEnableEpochsHandler)
		require.NoError(t, err)
		require.False(t, check.IfNil(kvs))
		require.Equal(t, funcGasCost, kvs.funcGasCost)
		require.Equal(t, gasConfig, kvs.gasConfig)
	})
}

func TestSaveKeyValueStorageFunc_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	funcGasCost := uint64(1)
	gasConfig := vmcommon.BaseOperationCost{
		StorePerByte: 1,
	}

	kvs, _ := NewSaveKeyValueStorageFunc(gasConfig, funcGasCost, disabledFixForSaveKeyValueEnableEpochsHandler)
	require.NotNil(t, kvs)

	newGasConfig := vmcommon.BaseOperationCost{
		StorePerByte: 37,
	}
	newGasCost := &vmcommon.GasCost{BaseOperationCost: newGasConfig}

	kvs.SetNewGasConfig(newGasCost)

	require.Equal(t, newGasConfig, kvs.gasConfig)
}

func TestSaveKeyValue_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	funcGasCost := uint64(1)
	gasConfig := vmcommon.BaseOperationCost{
		StorePerByte:      1,
		ReleasePerByte:    1,
		DataCopyPerByte:   1,
		PersistPerByte:    1,
		CompilePerByte:    1,
		AoTPreparePerByte: 1,
	}

	skv, _ := NewSaveKeyValueStorageFunc(gasConfig, funcGasCost, disabledFixForSaveKeyValueEnableEpochsHandler)

	addr := []byte("addr")
	acc := mock.NewUserAccount(addr)
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  addr,
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
		RecipientAddr: addr,
	}

	_, err := skv.ProcessBuiltinFunction(acc, nil, vmInput)
	require.Equal(t, ErrInvalidArguments, err)

	_, err = skv.ProcessBuiltinFunction(nil, acc, nil)
	require.Equal(t, ErrNilVmInput, err)

	key := []byte("key")
	value := []byte("value")
	vmInput.Arguments = [][]byte{key, value}

	_, err = skv.ProcessBuiltinFunction(acc, nil, vmInput)
	require.Equal(t, ErrNilSCDestAccount, err)

	_, err = skv.ProcessBuiltinFunction(acc, acc, vmInput)
	require.Nil(t, err)
	retrievedValue, _, _ := acc.AccountDataHandler().RetrieveValue(key)
	require.True(t, bytes.Equal(retrievedValue, value))

	vmInput.CallerAddr = []byte("other")
	_, err = skv.ProcessBuiltinFunction(acc, acc, vmInput)
	require.True(t, errors.Is(err, ErrOperationNotPermitted))

	key = []byte(core.ProtectedKeyPrefix + "is the king")
	value = []byte("value")
	vmInput.Arguments = [][]byte{key, value}

	_, err = skv.ProcessBuiltinFunction(acc, acc, vmInput)
	require.True(t, errors.Is(err, ErrOperationNotPermitted))
}

func TestSaveKeyValue_ProcessBuiltinFunctionGetNodeFromDbKey(t *testing.T) {
	t.Parallel()

	funcGasCost := uint64(1)
	gasConfig := vmcommon.BaseOperationCost{
		StorePerByte:      1,
		ReleasePerByte:    1,
		DataCopyPerByte:   1,
		PersistPerByte:    1,
		CompilePerByte:    1,
		AoTPreparePerByte: 1,
	}

	skv, _ := NewSaveKeyValueStorageFunc(gasConfig, funcGasCost, disabledFixForSaveKeyValueEnableEpochsHandler)
	addr := []byte("addr")
	acc := &mock.AccountWrapMock{
		RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
			return nil, 0, core.NewGetNodeFromDBErrWithKey([]byte("key"), errors.New("error"), "")
		},
	}

	key := []byte("key")
	value := []byte("value")
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  addr,
			GasProvided: 50,
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{key, value},
		},
		RecipientAddr: addr,
	}

	_, err := skv.ProcessBuiltinFunction(acc, acc, vmInput)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestSaveKeyValueStorage_ProcessBuiltinFunctionNilAccountSender(t *testing.T) {
	t.Parallel()

	funcGasCost := uint64(1)
	gasConfig := vmcommon.BaseOperationCost{
		StorePerByte:      1,
		ReleasePerByte:    1,
		DataCopyPerByte:   1,
		PersistPerByte:    1,
		CompilePerByte:    1,
		AoTPreparePerByte: 1,
	}

	skv, _ := NewSaveKeyValueStorageFunc(gasConfig, funcGasCost, disabledFixForSaveKeyValueEnableEpochsHandler)

	addr := []byte("addr")
	acc := mock.NewUserAccount(addr)
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  addr,
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
		RecipientAddr: addr,
	}

	key := []byte("key")
	value := []byte("value")
	vmInput.Arguments = [][]byte{key, value}

	_, err := skv.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)
	retrievedValue, _, _ := acc.AccountDataHandler().RetrieveValue(key)
	require.True(t, bytes.Equal(retrievedValue, value))
}

func TestSaveKeyValue_ProcessBuiltinFunctionMultipleKeys(t *testing.T) {
	t.Run("should work with disabled fix", func(t *testing.T) {
		processBuiltinFunctionMultipleKeys(t, disabledFixForSaveKeyValueEnableEpochsHandler)
	})
	t.Run("should work with enabled fix", func(t *testing.T) {
		processBuiltinFunctionMultipleKeys(t, enabledFixForSaveKeyValueEnableEpochsHandler)
	})
}

func processBuiltinFunctionMultipleKeys(t *testing.T, enableEpochHandler vmcommon.EnableEpochsHandler) {
	t.Parallel()

	funcGasCost := uint64(1)
	gasConfig := vmcommon.BaseOperationCost{
		StorePerByte:    1,
		ReleasePerByte:  1,
		DataCopyPerByte: 1,
		PersistPerByte:  1,
		CompilePerByte:  1,
	}
	skv, _ := NewSaveKeyValueStorageFunc(gasConfig, funcGasCost, enableEpochHandler)

	addr := []byte("addr")
	acc := mock.NewUserAccount(addr)
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  addr,
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
		RecipientAddr: addr,
	}

	key := []byte("key")
	value := []byte("value")
	vmInput.Arguments = [][]byte{key, value, key, value, key}

	_, err := skv.ProcessBuiltinFunction(acc, nil, vmInput)
	require.Equal(t, err, ErrInvalidArguments)

	key2 := []byte("key2")
	value2 := []byte("value2")
	vmInput.Arguments = [][]byte{key, value, key2, value2}

	_, err = skv.ProcessBuiltinFunction(acc, acc, vmInput)
	require.Nil(t, err)
	retrievedValue, _, _ := acc.AccountDataHandler().RetrieveValue(key)
	require.True(t, bytes.Equal(retrievedValue, value))
	retrievedValue, _, _ = acc.AccountDataHandler().RetrieveValue(key2)
	require.True(t, bytes.Equal(retrievedValue, value2))

	vmInput.GasProvided = 1
	vmInput.Arguments = [][]byte{[]byte("key3"), []byte("value")}
	_, err = skv.ProcessBuiltinFunction(acc, acc, vmInput)
	require.Equal(t, err, ErrNotEnoughGas)
}

func TestSaveKeyValue_ProcessBuiltinFunctionExistingKeyAndNotEnoughGas(t *testing.T) {
	t.Parallel()

	funcGasCost := uint64(100)
	persistPerByte := uint64(5)
	gasConfig := vmcommon.BaseOperationCost{
		StorePerByte:    1,
		ReleasePerByte:  1,
		DataCopyPerByte: 1,
		PersistPerByte:  persistPerByte,
		CompilePerByte:  1,
	}

	addr := []byte("addr")
	acc := mock.NewUserAccount(addr)
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  addr,
			GasProvided: 0,
			CallValue:   big.NewInt(0),
		},
		RecipientAddr: addr,
	}

	key := []byte("key")
	value := []byte("value")
	vmInput.Arguments = [][]byte{key, value}

	acc.Storage[string(key)] = value

	t.Run("backward compatibility: do not return error but a negative gas remaining value", func(t *testing.T) {
		skv, _ := NewSaveKeyValueStorageFunc(gasConfig, funcGasCost, disabledFixForSaveKeyValueEnableEpochsHandler)
		vmOutput, err := skv.ProcessBuiltinFunction(acc, acc, vmInput)
		assert.Nil(t, err)
		expectedGasRemaining := 0 - funcGasCost - persistPerByte*uint64(len(key)+len(value))
		assert.Equal(t, expectedGasRemaining, vmOutput.GasRemaining) // overflow on uint64 occurs here
	})
	t.Run("should return not enough of gas if the fix is enabled", func(t *testing.T) {
		skv, _ := NewSaveKeyValueStorageFunc(gasConfig, funcGasCost, enabledFixForSaveKeyValueEnableEpochsHandler)
		vmOutput, err := skv.ProcessBuiltinFunction(acc, acc, vmInput)
		assert.Equal(t, ErrNotEnoughGas, err)
		assert.Nil(t, vmOutput)
	})
}
