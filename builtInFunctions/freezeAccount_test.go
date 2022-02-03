package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	guardiansData "github.com/ElrondNetwork/elrond-go-core/data/guardians"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/require"
)

func requireAccountFrozen(t *testing.T, account vmcommon.UserAccountHandler, frozen bool) {
	codeMetaDataBytes := account.GetCodeMetadata()
	codeMetaData := vmcommon.CodeMetadataFromBytes(codeMetaDataBytes)

	require.Equal(t, frozen, codeMetaData.Frozen)
}

func TestNewFreezeAccountFunc(t *testing.T) {
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
			expectedErr: ErrNilMarshaller,
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
			require.Equal(t, core.BuiltInFunctionFreezeAccount, freezeAccountFunc.function)
			require.Equal(t, core.BuiltInFunctionUnfreezeAccount, unFreezeAccountFunc.function)
		}
	}
}

func TestFreezeAccount_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()
	freezeAccountFunc, _ := NewUnfreezeAccountFunc(args)
	require.Equal(t, args.FuncGasCost, freezeAccountFunc.funcGasCost)

	newFreezeAccountCost := args.FuncGasCost + 1
	newGasCost := &vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{FreezeAccount: newFreezeAccountCost}}

	freezeAccountFunc.SetNewGasConfig(newGasCost)
	require.Equal(t, newFreezeAccountCost, freezeAccountFunc.funcGasCost)
}

func TestFreezeUnfreezeAccountFunc_ProcessBuiltinFunctionAccountsAlreadyHaveFrozenBitSetExpectError(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()

	enabledGuardian := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch - 1,
	}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian}}

	dataTrie := &mock.DataTrieTrackerStub{
		RetrieveValueCalled: func(key []byte) ([]byte, error) {
			return args.Marshaller.Marshal(guardians)
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
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)
	freezeAccountFunc.EpochConfirmed(currentEpoch, 0)

	unfreezeAccountFunc, _ := NewUnfreezeAccountFunc(args)
	unfreezeAccountFunc.EpochConfirmed(currentEpoch, 0)

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

func TestFreezeAccount_ProcessBuiltinFunctionInvalidArgExpectError(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()
	freezeAccountFunc, _ := NewUnfreezeAccountFunc(args)

	vmInput := getDefaultVmInput([][]byte{})
	vmInput.CallValue = big.NewInt(1)
	account := mock.NewUserAccount(userAddress)

	output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)

}

func TestFreezeAccount_ProcessBuiltinFunctionCannotGetGuardianExpectError(t *testing.T) {
	t.Parallel()

	errRetrieveVal := errors.New("error retrieving value for key")
	accountHandler := &mock.DataTrieTrackerStub{
		RetrieveValueCalled: func(key []byte) ([]byte, error) {
			return nil, errRetrieveVal
		},
	}
	wasAccountAltered := atomic.Flag{}
	account := &mock.UserAccountStub{
		Address: userAddress,
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return accountHandler
		},
		SetCodeMetaDataCalled: func([]byte) {
			wasAccountAltered.SetValue(true)
		},
	}

	args := createFreezeAccountArgs()
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)
	vmInput := getDefaultVmInput([][]byte{})

	output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Equal(t, errRetrieveVal, err)
	require.False(t, wasAccountAltered.IsSet())
}

func TestFreezeAccount_ProcessBuiltinFunctionFreezeAccountNoEnabledGuardianExpectError(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()

	pendingGuardian := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch + 1,
	}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{pendingGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	requireAccountFrozen(t, account, false)

	vmInput := getDefaultVmInput([][]byte{})
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)

	output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Equal(t, ErrNoGuardianEnabled, err)
	requireAccountFrozen(t, account, false)
}

func TestFreezeAccount_ProcessBuiltinFunctionUnfreeze(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()
	enabledGuardian := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch - 1,
	}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	code := vmcommon.CodeMetadata{Frozen: true}
	account.SetCodeMetadata(code.ToBytes())
	requireAccountFrozen(t, account, true)

	vmInput := getDefaultVmInput([][]byte{})
	freezeAccountFunc, _ := NewUnfreezeAccountFunc(args)
	freezeAccountFunc.EpochConfirmed(currentEpoch, 0)

	output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)
	requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)
	requireAccountFrozen(t, account, false)
}

func TestFreezeAccount_ProcessBuiltinFunctionFreeze(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()
	enabledGuardian := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch - 1,
	}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	requireAccountFrozen(t, account, false)

	vmInput := getDefaultVmInput([][]byte{})
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)
	freezeAccountFunc.EpochConfirmed(currentEpoch, 0)

	output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)
	requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)
	requireAccountFrozen(t, account, true)
}

func createFreezeAccountArgs() FreezeAccountArgs {
	return FreezeAccountArgs{
		BaseAccountFreezerArgs:   createBaseAccountFreezerArgs(),
		FreezeAccountEnableEpoch: 1000,
	}
}
