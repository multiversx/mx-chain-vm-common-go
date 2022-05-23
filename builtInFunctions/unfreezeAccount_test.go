package builtInFunctions

import (
	"errors"
	"strings"
	"testing"

	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	mockvm "github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/require"
)

func TestUnfreezeAccountFunc_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()
	vmInput := getDefaultVmInput([][]byte{})

	t.Run("invalid args, expect error", func(t *testing.T) {
		unfreezeAccountFunc, _ := NewUnfreezeAccountFunc(args)
		unfreezeAccountFunc.EpochConfirmed(currentEpoch, 0)
		output, err := unfreezeAccountFunc.ProcessBuiltinFunction(nil, nil, vmInput)
		require.Nil(t, output)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), ErrNilUserAccount.Error()))
	})

	t.Run("account has no enabled guardian, expect error", func(t *testing.T) {
		expectedErr := errors.New("expected err")
		args.GuardedAccountHandler = &mockvm.GuardedAccountHandlerStub{
			GetActiveGuardianCalled: func(handler vmcommon.UserAccountHandler) ([]byte, error) {
				return nil, expectedErr
			},
		}
		unfreezeAccountFunc, _ := NewUnfreezeAccountFunc(args)
		unfreezeAccountFunc.EpochConfirmed(currentEpoch, 0)
		address := generateRandomByteArray(pubKeyLen)
		account := mockvm.NewUserAccount(address)
		vmInput.CallerAddr = address

		code := vmcommon.CodeMetadata{Frozen: true}
		account.SetCodeMetadata(code.ToBytes())

		output, err := unfreezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, output)
		require.Equal(t, expectedErr, err)
		requireAccountFrozen(t, account, true)
	})

	t.Run("unfreeze account should work", func(t *testing.T) {
		args.GuardedAccountHandler = &mockvm.GuardedAccountHandlerStub{
			GetActiveGuardianCalled: func(handler vmcommon.UserAccountHandler) ([]byte, error) {
				return []byte("active Guardian"), nil
			},
		}
		unfreezeAccountFunc, _ := NewUnfreezeAccountFunc(args)
		unfreezeAccountFunc.EpochConfirmed(currentEpoch, 0)

		address := generateRandomByteArray(pubKeyLen)
		account := mockvm.NewUserAccount(address)
		vmInput.CallerAddr = address

		code := vmcommon.CodeMetadata{Frozen: true}
		account.SetCodeMetadata(code.ToBytes())
		requireAccountFrozen(t, account, true)

		output, err := unfreezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, err)
		requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)
		requireAccountFrozen(t, account, false)
	})
}
