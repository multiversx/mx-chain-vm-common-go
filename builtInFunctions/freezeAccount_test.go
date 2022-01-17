package builtInFunctions

import (
	"errors"
	"math/big"
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

func TestNewFreezeAccountFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args                func() FreezeAccountArgs
		expectedBuiltInFunc string
		expectedErr         error
	}{
		{
			args: func() FreezeAccountArgs {
				args := createFreezeAccountArgs(false)
				args.Marshaller = nil
				return args
			},
			expectedBuiltInFunc: "",
			expectedErr:         ErrNilMarshaller,
		},
		{
			args: func() FreezeAccountArgs {
				return createFreezeAccountArgs(true)
			},
			expectedBuiltInFunc: BuiltInFunctionFreezeAccount,
			expectedErr:         nil,
		},
		{
			args: func() FreezeAccountArgs {
				return createFreezeAccountArgs(false)
			},
			expectedBuiltInFunc: BuiltInFunctionUnfreezeAccount,
			expectedErr:         nil,
		},
	}

	for _, test := range tests {
		freezeAccountFunc, err := NewFreezeAccountFunc(test.args())
		if test.expectedErr != nil {
			require.Nil(t, freezeAccountFunc)
			require.Error(t, err)
			require.Equal(t, test.expectedErr, err)
		} else {
			require.Nil(t, err)
			require.NotNil(t, freezeAccountFunc)
			require.Equal(t, test.expectedBuiltInFunc, freezeAccountFunc.function)
		}
	}
}

func TestFreezeAccount_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs(false)
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)
	require.Equal(t, args.FuncGasCost, freezeAccountFunc.funcGasCost)

	newFreezeAccountCost := args.FuncGasCost + 1
	newGasCost := &vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{FreezeAccount: newFreezeAccountCost}}
	freezeAccountFunc.SetNewGasConfig(newGasCost)
	require.Equal(t, newFreezeAccountCost, freezeAccountFunc.funcGasCost)
}

func TestFreezeAccount_ProcessBuiltinFunctionInvalidArgExpectError(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs(false)
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)
	vmInput := getDefaultVmInput(BuiltInFunctionUnfreezeAccount, [][]byte{})
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

	args := createFreezeAccountArgs(true)
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)
	vmInput := getDefaultVmInput(BuiltInFunctionUnfreezeAccount, [][]byte{})
	output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Equal(t, errRetrieveVal, err)
	require.False(t, wasAccountAltered.IsSet())
}

func TestFreezeAccount_ProcessBuiltinFunction_Unfreeze_ExpectAccountUnfrozen(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs(false)
	enabledGuardian := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() - 1,
	}
	guardians := &Guardians{Data: []*Guardian{enabledGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	code := vmcommon.CodeMetadata{Frozen: true}
	account.SetCodeMetadata(code.ToBytes())
	requireAccountFrozen(t, account, true)

	vmInput := getDefaultVmInput(BuiltInFunctionFreezeAccount, [][]byte{})
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)
	output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)
	requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)
	requireAccountFrozen(t, account, false)
}

func TestFreezeAccount_ProcessBuiltinFunction_Freeze_AccountDoesNotHaveEnabledGuardian_ExpectError(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs(true)

	pendingGuardian := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() + 1,
	}
	guardians := &Guardians{Data: []*Guardian{pendingGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	vmInput := getDefaultVmInput(BuiltInFunctionFreezeAccount, [][]byte{})

	freezeAccountFunc, _ := NewFreezeAccountFunc(args)
	output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Equal(t, ErrNoGuardianEnabled, err)
	requireAccountFrozen(t, account, false)
}

func TestFreezeAccount_ProcessBuiltinFunction_AccountHasOneEnabledGuardian_ExpectAccountFrozen(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs(true)
	enabledGuardian := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() - 1,
	}
	guardians := &Guardians{Data: []*Guardian{enabledGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	vmInput := getDefaultVmInput(BuiltInFunctionFreezeAccount, [][]byte{})

	freezeAccountFunc, _ := NewFreezeAccountFunc(args)
	output, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)
	require.Nil(t, err)

	requireAccountFrozen(t, account, true)
}

func createFreezeAccountArgs(freeze bool) FreezeAccountArgs {
	baseArgs := createBaseAccountFreezerArgs()

	return FreezeAccountArgs{
		BaseAccountFreezerArgs:   baseArgs,
		Freeze:                   freeze,
		FreezeAccountEnableEpoch: 1000,
	}
}
