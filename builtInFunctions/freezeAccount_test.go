package builtInFunctions

import (
	"errors"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/require"
)

func requireAccountFrozen(t *testing.T, account vmcommon.UserAccountHandler, frozen bool) {
	codeMetaDataBytes := account.GetCodeMetadata()
	codeMetaData := vmcommon.CodeMetadataFromBytes(codeMetaDataBytes)

	require.Equal(t, frozen, codeMetaData.Frozen)
}

func TestNewFreezeAccountFuncAndNewUnfreezeAccountFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args        func() FreezeAccountArgs
		expectedErr error
	}{
		{
			args: func() FreezeAccountArgs {
				args := createFreezeAccountArgs()
				args.Marshaller = nil
				return args
			},
			expectedErr: ErrNilMarshalizer,
		},
		{
			args: func() FreezeAccountArgs {
				return createFreezeAccountArgs()
			},
			expectedErr: nil,
		},
		{
			args: func() FreezeAccountArgs {
				return createFreezeAccountArgs()
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		freezeAccountFunc, errFreeze := NewFreezeAccountFunc(test.args())
		unFreezeAccountFunc, errUnfreeze := NewUnfreezeAccountFunc(test.args())
		if test.expectedErr != nil {
			require.Nil(t, freezeAccountFunc)
			require.Nil(t, unFreezeAccountFunc)
			require.Equal(t, test.expectedErr, errFreeze)
			require.Equal(t, test.expectedErr, errUnfreeze)
		} else {
			require.Nil(t, errFreeze)
			require.Nil(t, errUnfreeze)
			require.NotNil(t, freezeAccountFunc)
			require.NotNil(t, unFreezeAccountFunc)
			require.NotEqual(t, freezeAccountFunc, unFreezeAccountFunc)
		}
	}
}

func TestFreezeUnfreezeAccountFunc_ProcessBuiltinFunctionAccountsAlreadyHaveFrozenBitSetExpectError(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()

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
			codeMetaData := vmcommon.CodeMetadata{Frozen: accountFrozen}
			return codeMetaData.ToBytes()
		},
	}

	vmInput := getDefaultVmInput([][]byte{})
	args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsSetGuardianEnabledField:   true,
		IsFreezeAccountEnabledField: true,
	}
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)
	unfreezeAccountFunc, _ := NewUnfreezeAccountFunc(args)

	t.Run("try to freeze frozen account, expected error", func(t *testing.T) {
		accountFrozen = true
		output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, output)
		require.Equal(t, ErrSetFreezeAccount, err)
		require.False(t, wasAccountAltered.IsSet())
	})

	t.Run("try to unfreeze unfrozen account, expect error", func(t *testing.T) {
		accountFrozen = false
		output, err := unfreezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, output)
		require.Equal(t, ErrSetUnfreezeAccount, err)
		require.False(t, wasAccountAltered.IsSet())
	})
}

func TestFreezeAccountFunc_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()
	vmInput := getDefaultVmInput([][]byte{})

	t.Run("invalid args, expect error", func(t *testing.T) {
		args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
			IsFreezeAccountEnabledField: true,
			IsSetGuardianEnabledField:   true,
		}
		freezeAccountFunc, _ := NewFreezeAccountFunc(args)
		output, err := freezeAccountFunc.ProcessBuiltinFunction(nil, nil, vmInput)
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
			IsFreezeAccountEnabledField: true,
			IsSetGuardianEnabledField:   true,
		}
		freezeAccountFunc, _ := NewFreezeAccountFunc(args)
		address := generateRandomByteArray(pubKeyLen)
		account := mock.NewUserAccount(address)
		requireAccountFrozen(t, account, false)

		vmInput.CallerAddr = account.Address
		output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, output)
		require.Equal(t, expectedErr, err)
		requireAccountFrozen(t, account, false)
		require.False(t, cleanCalled)
	})

	t.Run("freeze account should work", func(t *testing.T) {
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
			IsFreezeAccountEnabledField: true,
			IsSetGuardianEnabledField:   true,
		}
		freezeAccountFunc, _ := NewFreezeAccountFunc(args)
		address := generateRandomByteArray(pubKeyLen)
		account := mock.NewUserAccount(address)
		vmInput.CallerAddr = account.Address
		requireAccountFrozen(t, account, false)

		output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, err)
		requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)
		requireAccountFrozen(t, account, true)
		require.True(t, cleanCalled)
	})
}
