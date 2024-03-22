package builtInFunctions

import (
	"errors"
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/atomic"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func requireAccountFrozen(t *testing.T, account vmcommon.UserAccountHandler, frozen bool) {
	codeMetaDataBytes := account.GetCodeMetadata()
	codeMetaData := vmcommon.CodeMetadataFromBytes(codeMetaDataBytes)

	require.Equal(t, frozen, codeMetaData.Guarded)
}

func TestNewGuardAccountFuncAndNewUnGuardAccountFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args        func() GuardAccountArgs
		expectedErr error
	}{
		{
			args: func() GuardAccountArgs {
				args := createGuardAccountArgs()
				args.Marshaller = nil
				return args
			},
			expectedErr: ErrNilMarshalizer,
		},
		{
			args: func() GuardAccountArgs {
				return createGuardAccountArgs()
			},
			expectedErr: nil,
		},
		{
			args: func() GuardAccountArgs {
				return createGuardAccountArgs()
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		guardAccountFunc, errGuard := NewGuardAccountFunc(test.args())
		unGuardAccountFunc, errUnGuard := NewUnGuardAccountFunc(test.args())
		if test.expectedErr != nil {
			require.Nil(t, guardAccountFunc)
			require.Nil(t, unGuardAccountFunc)
			require.Equal(t, test.expectedErr, errGuard)
			require.Equal(t, test.expectedErr, errUnGuard)
		} else {
			require.Nil(t, errGuard)
			require.Nil(t, errUnGuard)
			require.NotNil(t, guardAccountFunc)
			require.NotNil(t, unGuardAccountFunc)
			require.NotEqual(t, guardAccountFunc, unGuardAccountFunc)
		}
	}
}

func TestGuardUnGuardAccountFunc_ProcessBuiltinFunctionAccountsAlreadyHaveGuardedBitSetExpectError(t *testing.T) {
	t.Parallel()

	args := createGuardAccountArgs()

	dataTrie := &mock.DataTrieTrackerStub{
		RetrieveValueCalled: func(key []byte) ([]byte, uint32, error) {
			return []byte("marshalled guardians data"), 0, nil
		},
	}

	var accountFrozen bool
	wasAccountAltered := &atomic.Flag{}
	account := &mock.UserAccountStub{
		Address: userAddress,
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return dataTrie
		},
		SetCodeMetaDataCalled: func([]byte) {
			wasAccountAltered.SetValue(true)
		},
		GetCodeMetaDataCalled: func() []byte {
			codeMetaData := vmcommon.CodeMetadata{Guarded: accountFrozen}
			return codeMetaData.ToBytes()
		},
	}

	vmInput := getDefaultVmInput([][]byte{})
	args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SetGuardianFlag
		},
	}
	guardAccountFunc, _ := NewGuardAccountFunc(args)
	unGuardAccountFunc, _ := NewUnGuardAccountFunc(args)

	t.Run("try to guard guarded account, expected error", func(t *testing.T) {
		accountFrozen = true
		output, err := guardAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, output)
		require.Equal(t, ErrSetGuardAccountFlag, err)
		require.False(t, wasAccountAltered.IsSet())
	})

	t.Run("try to un-guard un-guarded account, expect error", func(t *testing.T) {
		accountFrozen = false
		output, err := unGuardAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, output)
		require.Equal(t, ErrSetUnGuardAccount, err)
		require.False(t, wasAccountAltered.IsSet())
	})
}

func TestGuardAccountFunc_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	args := createGuardAccountArgs()
	vmInput := getDefaultVmInput([][]byte{})

	t.Run("invalid args, expect error", func(t *testing.T) {
		args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SetGuardianFlag
			},
		}
		guardAccountFunc, _ := NewGuardAccountFunc(args)
		output, err := guardAccountFunc.ProcessBuiltinFunction(nil, nil, vmInput)
		require.Nil(t, output)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), ErrNilUserAccount.Error()))
	})

	t.Run("account has no enabled guardian, expect error", func(t *testing.T) {
		expectedErr := errors.New("expected err")
		cleanCalled := false
		args.GuardedAccountHandler = &mock.GuardedAccountHandlerStub{
			GetActiveGuardianCalled: func(handler vmcommon.UserAccountHandler) ([]byte, error) {
				return nil, expectedErr
			},
			CleanOtherThanActiveCalled: func(uah vmcommon.UserAccountHandler) {
				cleanCalled = true
			},
		}

		args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SetGuardianFlag
			},
		}
		guardAccountFunc, _ := NewGuardAccountFunc(args)
		address := generateRandomByteArray(pubKeyLen)
		account := mock.NewUserAccount(address)
		requireAccountFrozen(t, account, false)

		vmInput.CallerAddr = account.Address
		vmInput.RecipientAddr = account.Address
		output, err := guardAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, output)
		require.Equal(t, expectedErr, err)
		requireAccountFrozen(t, account, false)
		require.False(t, cleanCalled)
	})

	t.Run("guard account should work", func(t *testing.T) {
		cleanCalled := false
		args.GuardedAccountHandler = &mock.GuardedAccountHandlerStub{
			GetActiveGuardianCalled: func(handler vmcommon.UserAccountHandler) ([]byte, error) {
				return []byte("active guardian"), nil
			},
			CleanOtherThanActiveCalled: func(uah vmcommon.UserAccountHandler) {
				cleanCalled = true
			},
		}

		args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SetGuardianFlag
			},
		}
		guardAccountFunc, _ := NewGuardAccountFunc(args)
		address := generateRandomByteArray(pubKeyLen)
		account := mock.NewUserAccount(address)
		vmInput.CallerAddr = account.Address
		vmInput.RecipientAddr = account.Address
		requireAccountFrozen(t, account, false)

		output, err := guardAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, err)
		entry := &vmcommon.LogEntry{
			Address:    address,
			Identifier: []byte(core.BuiltInFunctionGuardAccount),
		}
		requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost, entry)
		requireAccountFrozen(t, account, true)
		require.True(t, cleanCalled)
	})
}
