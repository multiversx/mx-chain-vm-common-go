package builtInFunctions

import (
	"errors"
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	mockvm "github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestUnGuardAccountFunc_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	args := createGuardAccountArgs()
	vmInput := getDefaultVmInput([][]byte{})

	t.Run("invalid args, expect error", func(t *testing.T) {
		args.EnableEpochsHandler = &mockvm.EnableEpochsHandlerStub{
			IsSetGuardianEnabledField: true,
		}
		unGuardAccountFunc, _ := NewUnGuardAccountFunc(args)
		output, err := unGuardAccountFunc.ProcessBuiltinFunction(nil, nil, vmInput)
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
		args.EnableEpochsHandler = &mockvm.EnableEpochsHandlerStub{
			IsSetGuardianEnabledField: true,
		}
		unGuardAccountFunc, _ := NewUnGuardAccountFunc(args)
		address := generateRandomByteArray(pubKeyLen)
		account := mockvm.NewUserAccount(address)
		vmInput.CallerAddr = address
		vmInput.RecipientAddr = address

		code := vmcommon.CodeMetadata{Guarded: true}
		account.SetCodeMetadata(code.ToBytes())

		output, err := unGuardAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, output)
		require.Equal(t, expectedErr, err)
		requireAccountFrozen(t, account, true)
	})

	t.Run("un-guard account should work", func(t *testing.T) {
		args.GuardedAccountHandler = &mockvm.GuardedAccountHandlerStub{
			GetActiveGuardianCalled: func(handler vmcommon.UserAccountHandler) ([]byte, error) {
				return []byte("active Guardian"), nil
			},
		}
		args.EnableEpochsHandler = &mockvm.EnableEpochsHandlerStub{
			IsSetGuardianEnabledField: true,
		}
		unGuardAccountFunc, _ := NewUnGuardAccountFunc(args)

		address := generateRandomByteArray(pubKeyLen)
		account := mockvm.NewUserAccount(address)
		vmInput.CallerAddr = address
		vmInput.RecipientAddr = address

		code := vmcommon.CodeMetadata{Guarded: true}
		account.SetCodeMetadata(code.ToBytes())
		requireAccountFrozen(t, account, true)

		output, err := unGuardAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, err)

		entry := &vmcommon.LogEntry{
			Address:    address,
			Identifier: []byte(core.BuiltInFunctionUnGuardAccount),
		}
		requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost, entry)
		requireAccountFrozen(t, account, false)
	})
}
