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
	"github.com/stretchr/testify/require"
)

func TestNewRemoveCodeLeafFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil enable epochs handler", func(t *testing.T) {
		t.Parallel()

		rcl, err := NewMigrateCodeLeafFunc(10, nil, &mock.AccountsStub{})
		require.Nil(t, rcl)
		require.Equal(t, ErrNilEnableEpochsHandler, err)
	})

	t.Run("nil accounts db", func(t *testing.T) {
		t.Parallel()

		rcl, err := NewMigrateCodeLeafFunc(10, &mock.EnableEpochsHandlerStub{}, nil)
		require.Nil(t, rcl)
		require.Equal(t, ErrNilAccountsAdapter, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		enableEpochs := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == RemoveCodeLeafFlag
			},
		}

		rcl, err := NewMigrateCodeLeafFunc(10, enableEpochs, &mock.AccountsStub{})
		require.Nil(t, err)
		require.False(t, check.IfNil(rcl))

		require.True(t, rcl.IsActive())

		enableEpochs.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
			return false
		}

		require.False(t, rcl.IsActive())
	})
}

func TestRemoveCodeLeaf_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	t.Run("nil vm input", func(t *testing.T) {
		t.Parallel()

		rcl, _ := NewMigrateCodeLeafFunc(10, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := rcl.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), nil)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrNilVmInput, err)
	})

	t.Run("invalid num of args", func(t *testing.T) {
		t.Parallel()

		gasCost := uint64(10)

		rcl, _ := NewMigrateCodeLeafFunc(gasCost, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})

		addr := []byte("addr")
		key := []byte("codeHash")

		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr:  addr,
				GasProvided: 50,
				Arguments:   [][]byte{key},
				CallValue:   big.NewInt(0),
			},
			RecipientAddr: addr,
		}

		vmOutput, err := rcl.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), vmInput)
		require.Nil(t, vmOutput)
		require.True(t, errors.Is(err, ErrInvalidNumberOfArguments))
	})

	t.Run("should not call with value", func(t *testing.T) {
		t.Parallel()

		gasCost := uint64(10)
		rcl, _ := NewMigrateCodeLeafFunc(gasCost, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})

		addr := []byte("addr")
		key := []byte("codeHash")

		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr:  addr,
				GasProvided: 50,
				Arguments:   [][]byte{key},
				CallValue:   big.NewInt(2),
			},
			RecipientAddr: addr,
		}

		vmOutput, err := rcl.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), vmInput)
		require.Nil(t, vmOutput)
		require.True(t, errors.Is(err, ErrInvalidNumberOfArguments))
	})

	t.Run("should not call with value", func(t *testing.T) {
		t.Parallel()

		gasCost := uint64(10)
		rcl, _ := NewMigrateCodeLeafFunc(gasCost, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})

		addr := []byte("addr")

		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr:  addr,
				GasProvided: 50,
				CallValue:   big.NewInt(2),
			},
			RecipientAddr: addr,
		}

		vmOutput, err := rcl.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), vmInput)
		require.Nil(t, vmOutput)
		require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		wasCalled := false
		accounts := &mock.AccountsStub{
			GetCodeCalled: func(b []byte) []byte {
				return []byte("key code")
			},
			MigrateCodeLeafCalled: func(codeHash []byte) error {
				wasCalled = true
				return nil
			},
		}

		gasCost := uint64(10)

		rcl, _ := NewMigrateCodeLeafFunc(gasCost, &mock.EnableEpochsHandlerStub{}, accounts)

		addr := []byte("addr")

		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr:  addr,
				GasProvided: 50,
				CallValue:   big.NewInt(0),
			},
			RecipientAddr: addr,
		}

		vmOutput, err := rcl.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), vmInput)
		require.Nil(t, err)
		require.NotNil(t, vmOutput)
		require.Equal(t, uint64(40), vmOutput.GasRemaining)

		require.True(t, wasCalled)
	})

}

func TestRemoveCodeLeaf_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	rcl, err := NewMigrateCodeLeafFunc(10, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
	require.Nil(t, err)

	require.Equal(t, uint64(10), rcl.gasCost)

	rcl.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{MigrateCodeLeaf: 20}})

	require.Equal(t, uint64(20), rcl.gasCost)
}
