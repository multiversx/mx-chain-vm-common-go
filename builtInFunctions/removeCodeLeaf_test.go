package builtInFunctions

import (
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

		rcl, err := NewRemoveCodeLeafFunc(10, nil, &mock.AccountsStub{})
		require.Nil(t, rcl)
		require.Equal(t, ErrNilEnableEpochsHandler, err)
	})

	t.Run("nil accounts db", func(t *testing.T) {
		t.Parallel()

		rcl, err := NewRemoveCodeLeafFunc(10, &mock.EnableEpochsHandlerStub{}, nil)
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

		rcl, err := NewRemoveCodeLeafFunc(10, enableEpochs, &mock.AccountsStub{})
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

		rcl, err := NewRemoveCodeLeafFunc(10, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := rcl.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), nil)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrNilVmInput, err)
	})

}

func TestRemoveCodeLeaf_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	rcl, err := NewRemoveCodeLeafFunc(10, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
	require.Nil(t, err)

	require.Equal(t, uint64(10), rcl.gasCost)

	rcl.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{RemoveCodeLeaf: 20}})

	require.Equal(t, uint64(20), rcl.gasCost)
}
